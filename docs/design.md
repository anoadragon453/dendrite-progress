## Design

The site works by pulling data from Dendrite's
[testfile](https://github.com/matrix-org/dendrite/blob/master/testfile) which
tracks which [Sytest](https://github.com/matrix-org/sytest) tests Dendrite is
currently passing. It then compares this against the total number of SyTest
tests (found by grepping through SyTest's codebase) and converting that to a
ratio. The site then uses pretty charts to present that information.