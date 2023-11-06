Feature: MongoDB binary backups with PITR

  Background: Wait for working infrastructure
    Given prepared infrastructure
    And a configured s3 on minio01
    And mongodb initialized on mongodb01
    And oplog archiving is enabled on mongodb01
    And at least one oplog archive exists in storage

  Scenario: Binary backups, restores and deletes with pitr was done successfully
    Given mongodb01 has test mongodb data test1
    When we create binary mongo-backup on mongodb01
    Then we got 1 backup entries of mongodb01

    # First load
    Given mongodb01 has been loaded with "load1"
    And we save last oplog timestamp on mongodb01 to "after first load"
    And we save mongodb01 data "after first load"

    # Second backup was done successfully
    When we create binary mongo-backup on mongodb01
    Then we got 2 backup entries of mongodb01

    # Second load
    Given mongodb01 has been loaded with "load2"
    And we save last oplog timestamp on mongodb01 to "after second load"
    And we save mongodb01 data "after second load"

    #: Third load
    Given mongodb01 has been loaded with "load3"
    And we save last oplog timestamp on mongodb01 to "after third load"
    And we save mongodb01 data "after third load"

    # PITR: 1st backup to 1st ts
    Given mongodb02 has no data
    And mongodb initialized on mongodb02

    When we restore binary mongo-backup #0 to mongodb02
    And we restore from #0 backup to "after first load" timestamp to mongodb02
    And we save mongodb02 data "restore to after first load from second backup"
    Then we have same data in "after first load" and "restore to after first load from second backup"

    # PITR: 2nd backup to 2nd ts
    Given mongodb02 has no data
    And mongodb initialized on mongodb02

    When we restore binary mongo-backup #1 to mongodb02
    And we restore from #1 backup to "after second load" timestamp to mongodb02
    And we save mongodb02 data "restore to after second load from second backup"
    Then we have same data in "after second load" and "restore to after second load from second backup"

    # PITR: 1st backup to 2nd ts
    Given mongodb02 has no data
    And mongodb initialized on mongodb02

    When we restore binary mongo-backup #0 to mongodb02
    And we restore from #0 backup to "after second load" timestamp to mongodb02
    And we save mongodb02 data "restore to after second load from first backup"
    Then we have same data in "after second load" and "restore to after second load from first backup"

    # PITR: 2nd backup to 3rd ts
    Given mongodb02 has no data
    And mongodb initialized on mongodb02

    When we restore binary mongo-backup #1 to mongodb02
    And we restore from #1 backup to "after third load" timestamp to mongodb02
    And we save mongodb02 data "restore to after third load from first backup"
    Then we have same data in "after second load" and "restore to after second load from first backup"
