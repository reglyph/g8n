// Package fixtures holds the shared test schemas used by the language generator packages.
package fixtures

// BasicSchema exercises every field kind, defaults, sensitivity and required.
const BasicSchema = `# @package=env

# Host of the database
# @type=string @required
DB_HOST=db

# @type=port
# @default=5432
DB_PORT=

# @type=enum(dev,staging,prod)
# @default=dev
APP_ENV=

# @required @sensitive
API_KEY=

# @type=url
# @default=http://localhost
API_URL=

# @type=int64
# @default=1048576
MAX_UPLOAD_BYTES=

# @type=float64
# @default=0.5
JITTER=

# @type=bool
# @default=false
DEBUG=

# @type=string
# @default=hello
GREETING=

# @type=email
ADMIN_EMAIL=

OPTIONAL_NO_DEFAULT=

# @type=string
# @default=${DB_HOST}-primary
APP_URL=
`

// ConstraintsSchema exercises constraint emission (startsWith, regex) plus
// plain string/port/float fields without defaults.
const ConstraintsSchema = `# @package=env

# @type=string(startsWith=sk_)
# @default=sk_main
SERVICE=

# @type=string(regex=^[a-f0-9]{32}$)
TOKEN=

# @type=port
WEB_PORT=

# @type=string
NAME=

# @type=float64
RATIO=
`

// MinimalSchema is the smallest valid schema.
const MinimalSchema = `# @package=env
A=1
`
