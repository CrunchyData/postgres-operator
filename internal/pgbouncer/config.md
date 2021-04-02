<!--
 Copyright 2021 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->

PgBouncer is configured through INI files. It will reload these files when
receiving a `HUP` signal or [`RELOAD` command][RELOAD] in the admin console.

There is a [`SET` command][SET] available in the admin console, but it is not
clear when those changes take affect.

- https://www.pgbouncer.org/config.html

[RELOAD]: https://www.pgbouncer.org/usage.html#process-controlling-commands
[SET]: https://www.pgbouncer.org/usage.html#other-commands

The [`%include` directive](https://www.pgbouncer.org/config.html#include-directive)
allows one file to refer other (existing?) files.

There are three sections in the files:

 - `[pgbouncer]` is for settings that apply to the PgBouncer process.
 - `[databases]`
 - `[users]`

```
/etc/pgbouncer/pgbouncer.ini
/etc/pgbouncer/~postgres-operator.ini
/etc/pgbouncer/~postgres-operator/hba.conf
/etc/pgbouncer/~postgres-operator/users.txt
```
