locking
=======

Limits asynchronous processing of single Backup/Restore task, aggregated by `RequestedBackupAction` or `ScheduledBackup`.

**Background:**

There are multiple controllers running in parallel. Locking mechanism is making sure that single `RequestedBackupAction` or `ScheduledBackup` is processed
only by one controller at a time.

**Asynchronous processing:**

Still all controllers are asynchronous - multiple `RequestedBackupAction` or `ScheduledBackup` of different `.metadata.name` are processed in parallel.


**Reason:**

Avoid loop - one controller is updating `.status` field of our CRD, another one receives an update and starts processing, but the first controller still not finished processing,
and then triggers a next `.status` update, and other controller is picking an update event... that's crazy! That's why we limit the parallelism, to make it simpler.
