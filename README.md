# abp-cli

Go CLI for generating CRUD files in an ABP Framework application.

The generator detects the ABP solution/module under `src/`, detects the ABP Framework package version, previews every absolute file path, and asks for approval before writing files.

## Install

Recommended from a source checkout:

```bash
cd /path/to/abp-cli
sh ./install.sh
```

Install from GitHub with the setup script after the repository is public:

```bash
curl -fsSL https://raw.githubusercontent.com/mohamedhabibwork/abp-cli/main/install.sh | sh
```

Install into a specific bin directory:

```bash
INSTALL_DIR="$HOME/.local/bin" sh ./install.sh
```

Windows PowerShell:

```powershell
.\install.ps1
```

Install directly from GitHub after releases are published:

```bash
go install github.com/mohamedhabibwork/abp-cli@latest
```

Make sure the Go bin directory is on `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Verify:

```bash
abp-cli --help
abp-cli version
abp-cli generate --help
```

You can also run without installing:

```bash
go run ./scripts/abp_crud_generator.go --help
go run ./scripts/abp_crud_generator.go generate --help
```

More setup and release details are in [GitHub setup](docs/github-setup.md).

## Commands

```bash
abp-cli --help
abp-cli generate --help
abp-cli ui
abp-cli templates
abp-cli version
abp-cli install
abp-cli publish
abp-cli docs
```

`ui` opens an interactive CRUD wizard. `templates` exports editable generation templates. `version` prints release metadata. `install`, `publish`, and `docs` print copyable command documentation from the CLI.

## Generate CRUD

Generate from a JSON config file:

```bash
abp-cli generate \
  --config examples/product-crud.json \
  --root /path/to/abp-app \
  --mapping auto \
  --dry-run
```

When `--root` is omitted, the CLI starts from the current directory and walks upward until it finds an ABP solution with `src/*` layer projects. That means you can run it from the solution root or from a nested project folder such as `src/MyApp.Domain`. In multi-module solutions, running from inside a module project infers that current module automatically.

Open the interactive CLI UI:

```bash
abp-cli ui
```

The same wizard is available from `generate`:

```bash
abp-cli generate --ui
```

Preview planned file changes without writing:

```bash
abp-cli generate \
  --root /path/to/abp-app \
  --entity Product \
  --attr "string Name required maxlen:100" \
  --attr "decimal Price required" \
  --dry-run
```

Generate with interactive approval:

```bash
abp-cli generate \
  --root /path/to/abp-app \
  --entity Product \
  --attr "string Name required maxlen:100" \
  --attr "decimal Price required"
```

Generate without the approval prompt:

```bash
abp-cli generate \
  --root /path/to/abp-app \
  --entity Product \
  --attr "string Name required maxlen:100" \
  --attr "decimal Price required" \
  --yes
```

Generate only selected file groups or file types:

```bash
abp-cli generate \
  --root /path/to/abp-app \
  --entity Product \
  --files entity,dto,controller \
  --dry-run
```

You can also repeat `--file`:

```bash
abp-cli generate \
  --root /path/to/abp-app \
  --entity Product \
  --file domain \
  --file efcore \
  --yes
```

## Custom Templates

Export the built-in templates:

```bash
abp-cli templates --out ./abp-templates
```

Edit any exported template file, then generate with that directory:

```bash
abp-cli generate \
  --root /path/to/abp-app \
  --entity Product \
  --template-dir ./abp-templates \
  --dry-run
```

Only files present in `--template-dir` are overridden. Missing files automatically fall back to the built-in templates.

Template files use Go `text/template` syntax. Common fields include:

```text
Module
Entity
EntityType
KeyType
PluralEntity
Route
PermissionsClass
PermissionGroup
DbContextName
DbPropertiesName
Attrs
```

Pre-rendered blocks are also available for common generated code sections:

```text
EntityProperties
InputProperties
ReadDtoProperties
EtoProperties
GetListInputName
GetListInputProperties
FilterPredicates
ConstructorArgs
ConstructorSets
EntitySetters
UpdateArgs
AppServiceCreateArgs
AppServiceUpdateArgs
EFPropertyConfig
EFRelationshipConfig
```

Template filenames:

```text
appservice-interface.cs.tmpl
appservice.cs.tmpl
automapper-profile.cs.tmpl
constants.cs.tmpl
controller.cs.tmpl
create-dto.cs.tmpl
data-seeder.cs.tmpl
ef-configuration.cs.tmpl
ef-repository.cs.tmpl
entity-dto.cs.tmpl
entity.cs.tmpl
eto.cs.tmpl
event-types.cs.tmpl
get-list-input.cs.tmpl
idbcontext.cs.tmpl
localization.json.tmpl
many-to-many-join-configuration.cs.tmpl
many-to-many-join-entity.cs.tmpl
mapperly-mappers.cs.tmpl
permission-definition-provider.cs.tmpl
permissions.cs.tmpl
repository-interface.cs.tmpl
update-dto.cs.tmpl
```

The old direct flag style still works:

```bash
go run ./scripts/abp_crud_generator.go \
  --root /path/to/abp-app \
  --entity Product \
  --attr "string Name required maxlen:100"
```

## JSON Config

Use `--config` to load generation options from a JSON file:

```bash
abp-cli generate --config /path/to/product-crud.json
```

Command-line flags override values from the JSON file. This is useful when an example omits `root` and you want to point it at a real ABP app:

```bash
abp-cli generate \
  --config examples/product-crud.json \
  --root /path/to/abp-app \
  --dry-run
```

Structured attribute example:

```json
{
  "entity": "Product",
  "entityType": "FullAuditedAggregateRoot<Guid>",
  "key": "Guid",
  "files": "all",
  "attributes": [
    {
      "type": "string",
      "name": "Name",
      "required": true,
      "maxLength": 100,
      "filterable": true
    },
    {
      "type": "decimal",
      "name": "Price",
      "required": true,
      "filterable": true
    },
    {
      "type": "Guid",
      "name": "CategoryId",
      "required": true,
      "relation": {
        "kind": "reference",
        "entity": "Category",
        "inverse": "Products"
      }
    }
  ]
}
```

String attribute example:

```json
{
  "entity": "Category",
  "attributes": [
    "string Name required maxlen:100",
    "string Code maxlen:32"
  ]
}
```

Supported JSON fields:

```text
root
module
entity
entityType
entity_type
key
keyType
mapping
mappingStyle
templateDir
template_dir
template-dir
templates
templatesDir
templates_dir
overwrite
yes
assumeYes
dryRun
ui
interactive
files
file
fileTypes
file_types
file-types
types
only
attributes
attrs
attr
```

Attribute objects support:

```text
type
name
required
maxLength
max_length
maxlen
filterable
foreignKey
foreign_key
foreignkey
fk
relation
relationship
```

When `root` is omitted, it defaults to the current directory and then walks upward to find the ABP solution root. When `root` is present and relative, it is resolved relative to the JSON file and then uses the same upward detection.

Relation metadata supports `reference`, `collection`, and `manyToMany`. Collection relations require an explicit inverse navigation name. Many-to-many relations also require `joinEntity`, `thisKey`, and `otherKey`; the generator does not guess join-table names.

Example files are included in [examples](examples):

```text
examples/product-crud.json
examples/category-crud.json
examples/product-with-category-crud.json
```

## Attribute Syntax

Each `--attr` value uses this format:

```text
<csharp-type> <PropertyName> [required] [maxlen:N] [filterable] [foreignkey] [relation:Entity[:kind]]
```

Examples:

```bash
--attr "string Name required maxlen:100"
--attr "decimal Price required"
--attr "DateTime ReleaseDate"
--attr "Guid CategoryId required relation:Category"
```

## Options

```text
--root         ABP solution root. Defaults to current directory.
--module       ABP module namespace. Auto-detected when omitted.
--entity       Entity name in PascalCase.
--entity-type  ABP base type. Defaults to FullAuditedAggregateRoot<Guid>.
--key          Entity key type. Defaults to the generic type in --entity-type.
--mapping      Object mapping style: auto, automapper, or mapperly.
--template-dir Custom template directory. Matching files override built-ins.
--file         Generated file type or group to include. Repeatable.
--files        Comma-separated generated file types or groups to include.
--ui           Open the interactive CLI UI wizard.
--interactive  Alias for --ui.
--attr         Entity attribute. Repeat for each property.
--config       JSON config file with generation options.
--dry-run      Preview only.
--yes          Generate without approval prompt.
--overwrite    Overwrite generated files that already exist.
```

## Generated Files

The generator creates or updates files in these ABP layers:

```text
Domain.Shared
Domain
Application.Contracts
Application
HttpApi
EntityFrameworkCore
```

Typical generated surfaces include:

```text
Constants
Distributed event DTOs
Permissions and permission definition provider
Entity
DTOs
Custom Get<Entity>ListInput filter DTO
Application service interface
Application service
Repository interfaces and EF repository
AutoMapper profile or Mapperly mapper classes
HTTP API controller
Data seeder
EF Core entity configuration
Explicit many-to-many join entity and configuration
DbContext and IDbContext DbSet updates
Localization JSON keys
```

Use `all` to generate every supported CRUD surface. Use a group when you want a whole ABP layer:

```text
domain-shared
domain
contracts
application
httpapi
efcore
localization
```

Use individual file types when you want tighter control:

```text
constants
event-types
eto
entity
repository-interface
data-seeder
create-dto
update-dto
entity-dto
get-list-input
appservice-interface
permissions
permission-definition-provider
appservice
mapping
controller
ef-configuration
many-to-many-join-entity
many-to-many-join-configuration
ef-repository
dbcontext
idbcontext
localization-en
localization-ar
```

Useful aliases include `dto` for all DTO files including the list input, `repository` for domain and EF repositories, `permissions` for both permission files, and `dbcontexts` for both DbContext surfaces.

## ABP Version Compatibility

The CLI scans the ABP package version and mapping integration markers before planning files.

```text
ABP 8  Defaults to AutoMapper generation. CRUD services still use IRepository.
ABP 9  Defaults to AutoMapper generation unless Mapperly markers are present.
ABP 10 Defaults to Mapperly generation unless AutoMapper markers are present.
```

Mapping detection checks for package/module markers such as:

```text
Volo.Abp.AutoMapper
AbpAutoMapperModule
AddAutoMapperObjectMapper
Volo.Abp.Mapperly
AbpMapperlyModule
AddMapperlyObjectMapper
Riok.Mapperly
```

Override detection when needed:

```bash
abp-cli generate \
  --root /path/to/abp-app \
  --entity Product \
  --mapping mapperly \
  --attr "string Name required maxlen:100" \
  --dry-run
```

The preview prints the detected ABP version, selected mapping style, compatibility notes, and the full absolute path for every planned file.

More details are in [ABP version compatibility](docs/abp-version-compatibility.md).

## Publish

Initial GitHub repository setup:

```bash
git init
git add .
git commit -m "Initial ABP CLI"
git branch -M main
git remote add origin git@github.com:mohamedhabibwork/abp-cli.git
git push -u origin main
```

Common local commands:

```bash
make test
make build
make install
make release-check
abp-cli version
```

Release rules are based on Conventional Commits:

```text
BREAKING CHANGE or type!:  major version
feat:                     minor version
fix:, perf:, refactor:    patch version
build:, ci:, chore:       patch version
docs:, test:              patch version
```

Publish automatically from `main` or `master`:

```bash
git push origin main
```

The release workflow calculates the next version, creates the tag, builds release archives for macOS/Linux/Windows, generates release notes, and uploads checksums.

Every published version must pass `make release-check` first. If tests or smoke checks fail, the workflow stops before tag creation, archive build, or GitHub Release publishing.

Each published version also gets an automatic changelog entry in [CHANGELOG.md](CHANGELOG.md). Auto and manual branch releases commit the changelog before creating the release tag; direct `v*` tag pushes generate release notes and upload the changelog as a release asset without moving the existing tag.

Publish an explicit GitHub release by pushing a version tag:

```bash
go test ./...
git tag v0.1.0
git push origin v0.1.0
```

You can also run the release workflow manually in GitHub Actions and provide a version like `v0.2.0`.

Manual release binary build:

```bash
mkdir -p dist
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/abp-cli-darwin-arm64 .
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/abp-cli-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/abp-cli-windows-amd64.exe .
```

For public `go install` publishing, keep `go.mod` aligned with the final repository module path. If this repo moves, update it to that full path:

```text
module github.com/your-org/abp-cli
```

Then users can install a tagged release:

```bash
go install github.com/your-org/abp-cli@v0.1.0
```

Detailed repository setup is in [GitHub setup](docs/github-setup.md).
Release rules are in [.github/release-rules.md](.github/release-rules.md).

## Notes

- The generator prints full absolute paths before writing.
- Existing generated files are skipped unless `--overwrite` is passed.
- Permission and DbContext files are updated in place when detected.
- After generation, create the EF Core migration in the ABP app.
