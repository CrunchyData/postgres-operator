Exporter - AppendCustomQueries Enabled

Note: This series of tests depends on PGO being deployed with the AppendCustomQueries feature gate ON. There is a separate set of tests in e2e that tests exporter functionality without the AppendCustomQueries feature.

When running this test, make sure that the PGO_FEATURE_GATES environment variable is set to "AppendCustomQueries=true" on the PGO Deployment.
