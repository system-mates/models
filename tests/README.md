# Model acceptance tests

The acceptance suite starts a real Mate process with temporary instances built
from the models in this repository. It interacts with Mate through HTTP and
does not import runtime packages.

Run the suite from the models repository:

```sh
go test ./tests/acceptance -v
```

By default, the suite builds the sibling `../runtime` repository. To test a
specific runtime build or release candidate, provide its binary explicitly:

```sh
MATE_BINARY=/path/to/mate go test ./tests/acceptance -v
```

Every test gets its own mate directory, data directory, loopback listener,
cookie jar, and Mate process. Process output is included in the test log when a
scenario fails.

Current scenarios cover Logs with and without Users; Portal navigation,
resources, root-route ownership, and Users-aware presentation; the Users
bootstrap, login, administration, user creation, authorization, and logout
lifecycle, including role and password changes, service accounts, bearer-token
authorization, and token revocation; and Data Editor inspection and mutation of
deterministic State and Share values. Arch coverage includes its Operations,
Architecture, detail and static-resource surfaces, cross-model remembered
selection, authorization, and an administrative model restart observed through
persisted model state.

Tests select system models by name, such as `users`. Small models used solely
to exercise a system model belong under `tests/models` and are selected with a
`test/` prefix, such as `test/dataeditor-target`. Example models can be selected
with an `example/` prefix when their own supported behavior is under test. Add
future model scenarios to `tests/acceptance` and keep shared process and HTTP
behavior in `harness_test.go`.
