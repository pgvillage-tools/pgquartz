package etcd

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// Locker uses an etcd mutex to make sure only one instance of a job runs at a time.
type Locker struct {
	config     Config
	cli        *clientv3.Client
	session    *concurrency.Session
	mutex      *concurrency.Mutex
	context    context.Context
	cancelFunc context.CancelFunc
}

// NewEtcdLocker returns a Locker for the given config, with defaults applied.
func NewEtcdLocker(config Config) Locker {
	config.SetDefaults()
	return Locker{
		config: config,
	}
}

// Lock acquires the etcd lock, waiting at most the configured lock timeout.
func (el *Locker) Lock() {
	if el.config.LockKey == "" {
		log.Debug("lockKey not set")
		return
	}
	var err error
	log.Debug("starting etcd client")
	el.cli, err = clientv3.New(clientv3.Config{Endpoints: el.config.Endpoints})
	if err != nil {
		log.Fatal(err)
	}
	log.Debug("starting etcd session")
	// create a sessions to acquire a lock
	el.session, err = concurrency.NewSession(el.cli)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("getting mutex /pgquartz/%s/", el.config.LockKey)
	el.mutex = concurrency.NewMutex(el.session, fmt.Sprintf("/pgquartz/%s/", el.config.LockKey))
	// acquire lock, or wait to have it, but cancel wait after lockDuration
	el.context, el.cancelFunc = context.WithCancel(ctx)
	lockDuration, err := time.ParseDuration(el.config.LockTimeout)
	if err != nil {
		log.Fatal(err)
	}
	// We use AfterFunc here, because we want the lockDuration timeout to be cancelled if we have the lock
	// Inspired by https://stackoverflow.com/a/61455619
	t := time.AfterFunc(lockDuration, el.cancelFunc)
	log.Debug("locking mutex")
	if err := el.mutex.Lock(el.context); err != nil {
		log.Fatal(err)
	}
	// We have the lock. Let's stop the AfterFunc and not call cancelFunc anymore...
	log.Debug("stopping timer")
	t.Stop()
	// Relying on jobContext from hereon
}

// UnLock releases the etcd lock if it is held.
func (el *Locker) UnLock() {
	if el.mutex != nil {
		if err := el.mutex.Unlock(ctx); err != nil {
			log.Fatal(err)
		}
		el.mutex = nil
	}
}

// Close releases the lock and closes the etcd session and client.
func (el *Locker) Close() {
	log.Debug("unlocking")
	el.UnLock()
	log.Debug("cancelling context")
	if el.cancelFunc != nil {
		el.cancelFunc()
		el.cancelFunc = nil
	}
	log.Debug("closing session")
	if el.session != nil {
		_ = el.session.Close()
		el.session = nil
	}
	log.Debug("closing client")
	if el.cli != nil {
		_ = el.cli.Close()
		el.cli = nil
	}
}
