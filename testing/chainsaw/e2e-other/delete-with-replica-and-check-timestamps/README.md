This test originally existed as the second test-case in the `delete` KUTTL test.
The test as written was prone to occasional flakes, sometimes due to missing events
(which were being used to check the timestamp of the container delete event).

After discussion, we decided that this behavior (replica deleting before the primary)
was no longer required in v5, and the decision was made to sequester this test-case for
further testing and refinement.