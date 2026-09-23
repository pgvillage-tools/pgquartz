#!/bin/bash
echo "My PID is $$, ARGS are: $PGQ_INSTANCE_DELAY en $PGQ_INSTANCE_DESC" | tee -a /tmp/job3.txt
sleep "${PGQ_INSTANCE_DELAY}"
