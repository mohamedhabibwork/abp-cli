package abpcrud

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Options struct {
	Root        string
	Module      string
	Entity      string
	EntityType  string
	KeyType     string
	Mapping     string
	Files       []string
	TemplateDir string
	Interactive bool
	Overwrite   bool
	AssumeYes   bool
	DryRun      bool
	Attrs       []Attribute
}

type Attribute struct {
	Type       string
	Name       string
	Required   bool
	MaxLength  int
	ForeignKey bool
}

type AppInfo struct {
	Root                  string
	SolutionPath          string
	Module                string
	ABPVersion            string
	ABPVersionSource      string
	ABPMajor              int
	LayerDirs             map[string]string
	DbContextFile         string
	IDbContextFile        string
	DbContextName         string
	DbContextPrefix       string
	DbPropertiesName      string
	DbPropertiesNamespace string
	ResourceType          string
	ResourceNamespace     string
	MappingStyle          string
	MappingStyleSource    string
	DetectedModuleNames   []string
}

type Plan struct {
	Info    AppInfo
	Options Options
	Changes []FileChange
}

type FileChange struct {
	Type    string
	Path    string
	Action  string
	Content []byte
}

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ", ")
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

const (
	layerDomainShared         = "Domain.Shared"
	layerDomain               = "Domain"
	layerApplicationContracts = "Application.Contracts"
	layerApplication          = "Application"
	layerHttpAPI              = "HttpApi"
	layerEntityFrameworkCore  = "EntityFrameworkCore"
)

var layerNames = []string{
	layerDomainShared,
	layerDomain,
	layerApplicationContracts,
	layerApplication,
	layerHttpAPI,
	layerEntityFrameworkCore,
}

const defaultEntityType = "FullAuditedAggregateRoot<Guid>"

const (
	mappingAuto       = "auto"
	mappingAutoMapper = "automapper"
	mappingMapperly   = "mapperly"
)

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "generate", "gen", "crud":
			return runGenerate(args[1:], stdin, stdout, stderr)
		case "ui", "wizard", "interactive":
			nextArgs := append([]string{"--ui"}, args[1:]...)
			return runGenerate(nextArgs, stdin, stdout, stderr)
		case "templates", "template", "export-templates":
			return runTemplates(args[1:], stdout, stderr)
		case "version", "-v", "--version":
			PrintVersion(stdout)
			return 0
		case "install":
			PrintInstallDocs(stdout)
			return 0
		case "publish":
			PrintPublishDocs(stdout)
			return 0
		case "docs":
			PrintDocs(stdout)
			return 0
		case "help":
			PrintRootHelp(stdout)
			return 0
		case "-h", "--help":
			PrintRootHelp(stdout)
			return 0
		}
	}

	return runGenerate(args, stdin, stdout, stderr)
}

func runGenerate(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var rawAttrs stringList
	var rawFiles stringList
	var configPath string
	cliOpts := Options{
		Root:       ".",
		EntityType: defaultEntityType,
	}

	fs := flag.NewFlagSet("abp-crud-generator", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&configPath, "config", "", "JSON config file with generation options.")
	fs.StringVar(&cliOpts.Root, "root", ".", "ABP solution root. Defaults to the current directory.")
	fs.StringVar(&cliOpts.Module, "module", "", "ABP module/application namespace. Auto-detected from src/* layer projects when omitted.")
	fs.StringVar(&cliOpts.Entity, "entity", "", "Entity name in PascalCase, for example Product.")
	fs.StringVar(&cliOpts.EntityType, "entity-type", defaultEntityType, "ABP entity base type, for example FullAuditedAggregateRoot<Guid>, AggregateRoot<long>, Entity<Guid>, or ValueObject.")
	fs.StringVar(&cliOpts.KeyType, "key", "", "Entity key type. Defaults to the generic type in --entity-type.")
	fs.StringVar(&cliOpts.Mapping, "mapping", mappingAuto, "Object mapping style: auto, automapper, or mapperly.")
	fs.StringVar(&cliOpts.TemplateDir, "template-dir", "", "Directory containing custom templates. Files with matching names override built-in templates.")
	fs.Var(&rawFiles, "file", "Generated file type or group to include. Repeat or comma-separate. Use all, domain, contracts, application, httpapi, efcore, localization, entity, dto, controller, dbcontext, etc.")
	fs.Var(&rawFiles, "files", "Comma-separated generated file types or groups to include. Same values as --file.")
	fs.BoolVar(&cliOpts.Interactive, "ui", false, "Open the interactive CLI UI wizard before previewing file changes.")
	fs.BoolVar(&cliOpts.Interactive, "interactive", false, "Open the interactive CLI UI wizard before previewing file changes.")
	fs.BoolVar(&cliOpts.Overwrite, "overwrite", false, "Overwrite generated files that already exist.")
	fs.BoolVar(&cliOpts.AssumeYes, "yes", false, "Generate without asking for approval.")
	fs.BoolVar(&cliOpts.DryRun, "dry-run", false, "Show the detected app and planned file paths without writing.")
	fs.Var(&rawAttrs, "attr", `Entity attribute, repeatable. Example: --attr "string Name required maxlen:100"`)

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Generate ABP Framework CRUD files after previewing absolute file paths.")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, `  abp-cli generate --config examples/product-crud.json --dry-run`)
		fmt.Fprintln(stderr, `  abp-cli generate --root /path/to/app --entity Product --attr "string Name required maxlen:100" --attr "decimal Price required"`)
		fmt.Fprintln(stderr, `  abp-cli generate --root /path/to/app --entity Product --file entity --file dto --file controller`)
		fmt.Fprintln(stderr, `  abp-cli ui --root /path/to/app`)
		fmt.Fprintln(stderr, `  go run ./scripts/abp_crud_generator.go generate --root /path/to/app --entity Product --attr "string Name required maxlen:100"`)
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	cliAttrs, err := parseAttributeSpecs(rawAttrs)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	cliOpts.Attrs = cliAttrs
	cliOpts.Files = rawFiles

	opts := cliOpts
	if configPath != "" {
		configOpts, err := LoadOptionsFromJSON(configPath)
		if err != nil {
			fmt.Fprintf(stderr, "invalid --config %q: %v\n", configPath, err)
			return 2
		}
		opts = withOptionDefaults(configOpts)
		opts = applyCLIOverrides(opts, cliOpts, provided)
	}

	if opts.Interactive {
		reader := bufio.NewReader(stdin)
		var err error
		opts, err = RunInteractive(opts, reader, stdout)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		stdin = reader
	}

	plan, err := BuildPlan(opts)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	PrintPlan(stdout, plan)

	if opts.DryRun {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Dry run only. No files were written.")
		return 0
	}

	if !opts.AssumeYes {
		if !AskForApproval(stdin, stdout) {
			fmt.Fprintln(stdout, "Generation cancelled.")
			return 0
		}
	}

	if err := ApplyPlan(plan); err != nil {
		fmt.Fprintf(stderr, "generation failed: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "CRUD files generated successfully.")
	return 0
}

func parseAttributeSpecs(specs []string) ([]Attribute, error) {
	var attrs []Attribute
	for _, spec := range specs {
		attr, err := ParseAttribute(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid --attr %q: %w", spec, err)
		}
		attrs = append(attrs, attr)
	}
	return attrs, nil
}

func withOptionDefaults(opts Options) Options {
	if opts.Root == "" {
		opts.Root = "."
	}
	if opts.EntityType == "" {
		opts.EntityType = defaultEntityType
	}
	if opts.Mapping == "" {
		opts.Mapping = mappingAuto
	}
	return opts
}

func applyCLIOverrides(opts Options, cliOpts Options, provided map[string]bool) Options {
	if provided["root"] {
		opts.Root = cliOpts.Root
	}
	if provided["module"] {
		opts.Module = cliOpts.Module
	}
	if provided["entity"] {
		opts.Entity = cliOpts.Entity
	}
	if provided["entity-type"] {
		opts.EntityType = cliOpts.EntityType
	}
	if provided["key"] {
		opts.KeyType = cliOpts.KeyType
	}
	if provided["mapping"] {
		opts.Mapping = cliOpts.Mapping
	}
	if provided["template-dir"] {
		opts.TemplateDir = cliOpts.TemplateDir
	}
	if provided["file"] || provided["files"] {
		opts.Files = cliOpts.Files
	}
	if provided["ui"] || provided["interactive"] {
		opts.Interactive = cliOpts.Interactive
	}
	if provided["overwrite"] {
		opts.Overwrite = cliOpts.Overwrite
	}
	if provided["yes"] {
		opts.AssumeYes = cliOpts.AssumeYes
	}
	if provided["dry-run"] {
		opts.DryRun = cliOpts.DryRun
	}
	if provided["attr"] {
		opts.Attrs = cliOpts.Attrs
	}
	return opts
}

func PrintRootHelp(w io.Writer) {
	fmt.Fprintln(w, "ABP CRUD generator")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  generate   Detect an ABP app, preview CRUD file paths, and generate after approval")
	fmt.Fprintln(w, "  ui         Open the interactive CRUD generation wizard")
	fmt.Fprintln(w, "  templates  Export built-in templates for customization")
	fmt.Fprintln(w, "  version    Show version, commit, and build date")
	fmt.Fprintln(w, "  install    Show install commands")
	fmt.Fprintln(w, "  publish    Show build and release publishing commands")
	fmt.Fprintln(w, "  docs       Show short command documentation")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  abp-cli generate --config examples/product-crud.json --root /path/to/app --dry-run")
	fmt.Fprintln(w, `  abp-cli generate --root /path/to/app --entity Product --attr "string Name required maxlen:100"`)
	fmt.Fprintln(w, `  abp-cli generate --root /path/to/app --entity Product --files entity,dto,controller`)
	fmt.Fprintln(w, "  abp-cli ui")
	fmt.Fprintln(w, "  abp-cli templates --out ./abp-templates")
	fmt.Fprintln(w, "  abp-cli version")
	fmt.Fprintln(w, "  abp-cli install")
	fmt.Fprintln(w, "  abp-cli publish")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run `abp-cli generate --help` for generation options.")
}

func PrintInstallDocs(w io.Writer) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/path/to/abp-cli"
	}

	fmt.Fprintln(w, "Install from this source checkout:")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  cd %s\n", cwd)
	fmt.Fprintln(w, "  sh ./install.sh")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Install directly from GitHub after releases are published:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  go install github.com/mohamedhabibwork/abp-cli@latest")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Make sure the Go bin directory is on PATH:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, `  export PATH="$(go env GOPATH)/bin:$PATH"`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Verify:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  abp-cli --help")
	fmt.Fprintln(w, "  abp-cli version")
	fmt.Fprintln(w, "  abp-cli generate --help")
}

func PrintPublishDocs(w io.Writer) {
	fmt.Fprintln(w, "Build and publish commands:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  go test ./...")
	fmt.Fprintln(w, "  make build")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Automatic release rules:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  feat: changes create a minor release")
	fmt.Fprintln(w, "  fix:, perf:, refactor:, build:, ci:, chore:, docs:, and test: changes create a patch release")
	fmt.Fprintln(w, "  BREAKING CHANGE or ! in the commit type creates a major release")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "GitHub release examples:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  # Automatic version from conventional commits on main")
	fmt.Fprintln(w, "  git push origin main")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  # Explicit release tag")
	fmt.Fprintln(w, "  git tag v0.1.0")
	fmt.Fprintln(w, "  git push origin v0.1.0")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Before public go install publishing, make sure go.mod uses the final repository module path.")
}

func PrintDocs(w io.Writer) {
	PrintRootHelp(w)
	fmt.Fprintln(w)
	PrintInstallDocs(w)
	fmt.Fprintln(w)
	PrintPublishDocs(w)
}

func runTemplates(args []string, stdout, stderr io.Writer) int {
	var outputDir string
	var overwrite bool

	fs := flag.NewFlagSet("abp-crud-templates", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&outputDir, "out", "", "Directory where built-in templates should be written.")
	fs.StringVar(&outputDir, "output", "", "Directory where built-in templates should be written.")
	fs.BoolVar(&overwrite, "overwrite", false, "Overwrite existing template files.")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Export built-in templates so they can be customized and passed with --template-dir.")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  abp-cli templates --out ./abp-templates")
		fmt.Fprintln(stderr, "  abp-cli generate --template-dir ./abp-templates --entity Product")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if outputDir == "" {
		names, err := listDefaultTemplateNames()
		if err != nil {
			fmt.Fprintf(stderr, "failed to list templates: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "Built-in templates:")
		for _, name := range names {
			fmt.Fprintf(stdout, "  %s\n", name)
		}
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Export them with:")
		fmt.Fprintln(stdout, "  abp-cli templates --out ./abp-templates")
		return 0
	}

	changes, err := writeDefaultTemplates(outputDir, overwrite)
	if err != nil {
		fmt.Fprintf(stderr, "failed to export templates: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Templates exported to %s\n", outputDir)
	for _, change := range changes {
		fmt.Fprintf(stdout, "  %-13s %s\n", "["+change.Action+"]", change.Path)
	}
	return 0
}

func LoadOptionsFromJSON(path string) (Options, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Options{}, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return Options{}, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Options{}, err
	}

	opts := Options{
		Root:       getJSONText(raw, "root"),
		Module:     getJSONText(raw, "module"),
		Entity:     getJSONText(raw, "entity"),
		EntityType: firstNonEmpty(getJSONText(raw, "entityType"), getJSONText(raw, "entity_type"), getJSONText(raw, "entity-type")),
		KeyType:    firstNonEmpty(getJSONText(raw, "key"), getJSONText(raw, "keyType"), getJSONText(raw, "key_type")),
		Mapping:    firstNonEmpty(getJSONText(raw, "mapping"), getJSONText(raw, "mappingStyle"), getJSONText(raw, "mapping_style")),
		TemplateDir: firstNonEmpty(
			getJSONText(raw, "templateDir"),
			getJSONText(raw, "template_dir"),
			getJSONText(raw, "template-dir"),
			getJSONText(raw, "templates"),
			getJSONText(raw, "templatesDir"),
			getJSONText(raw, "templates_dir"),
		),
	}

	if value, ok := getJSONBool(raw, "overwrite"); ok {
		opts.Overwrite = value
	}
	if value, ok := getJSONBool(raw, "yes", "assumeYes", "assume_yes"); ok {
		opts.AssumeYes = value
	}
	if value, ok := getJSONBool(raw, "dryRun", "dry_run", "dry-run"); ok {
		opts.DryRun = value
	}
	if value, ok := getJSONBool(raw, "ui", "interactive"); ok {
		opts.Interactive = value
	}

	attrs, err := getJSONAttributes(raw)
	if err != nil {
		return Options{}, err
	}
	opts.Attrs = attrs
	files, err := getJSONStrings(raw, "files", "file", "fileTypes", "file_types", "file-types", "types", "only")
	if err != nil {
		return Options{}, err
	}
	opts.Files = files

	if opts.Root != "" && !filepath.IsAbs(opts.Root) {
		opts.Root = filepath.Join(filepath.Dir(absPath), opts.Root)
	}
	if opts.TemplateDir != "" && !filepath.IsAbs(opts.TemplateDir) {
		opts.TemplateDir = filepath.Join(filepath.Dir(absPath), opts.TemplateDir)
	}

	return opts, nil
}

func getJSONText(raw map[string]json.RawMessage, names ...string) string {
	for _, name := range names {
		value, ok := raw[name]
		if !ok {
			continue
		}
		var text string
		if err := json.Unmarshal(value, &text); err == nil {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func getJSONBool(raw map[string]json.RawMessage, names ...string) (bool, bool) {
	for _, name := range names {
		value, ok := raw[name]
		if !ok {
			continue
		}
		var parsed bool
		if err := json.Unmarshal(value, &parsed); err == nil {
			return parsed, true
		}
	}
	return false, false
}

func getJSONStrings(raw map[string]json.RawMessage, names ...string) ([]string, error) {
	var values []string
	for _, name := range names {
		value, ok := raw[name]
		if !ok {
			continue
		}

		trimmed := strings.TrimSpace(string(value))
		if strings.HasPrefix(trimmed, "[") {
			var items []string
			if err := json.Unmarshal(value, &items); err != nil {
				return nil, fmt.Errorf("%s must be a string array: %w", name, err)
			}
			values = append(values, items...)
			continue
		}

		var item string
		if err := json.Unmarshal(value, &item); err != nil {
			return nil, fmt.Errorf("%s must be a string or string array: %w", name, err)
		}
		values = append(values, item)
	}
	return values, nil
}

func getJSONAttributes(raw map[string]json.RawMessage) ([]Attribute, error) {
	values := make([]json.RawMessage, 0)
	for _, name := range []string{"attributes", "attrs", "attr"} {
		value, ok := raw[name]
		if !ok {
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(string(value)), "[") {
			var items []json.RawMessage
			if err := json.Unmarshal(value, &items); err != nil {
				return nil, fmt.Errorf("%s must be an array: %w", name, err)
			}
			values = append(values, items...)
			continue
		}

		values = append(values, value)
	}

	attrs := make([]Attribute, 0, len(values))
	for index, value := range values {
		attr, err := parseJSONAttribute(value)
		if err != nil {
			return nil, fmt.Errorf("attributes[%d]: %w", index, err)
		}
		attrs = append(attrs, attr)
	}
	return attrs, nil
}

func parseJSONAttribute(value json.RawMessage) (Attribute, error) {
	trimmed := strings.TrimSpace(string(value))
	if strings.HasPrefix(trimmed, `"`) {
		var spec string
		if err := json.Unmarshal(value, &spec); err != nil {
			return Attribute{}, err
		}
		return ParseAttribute(spec)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(value, &raw); err != nil {
		return Attribute{}, err
	}

	if spec := getJSONText(raw, "spec"); spec != "" {
		return ParseAttribute(spec)
	}

	attr := Attribute{
		Type:       getJSONText(raw, "type"),
		Name:       toPascal(getJSONText(raw, "name")),
		Required:   false,
		ForeignKey: false,
	}
	if value, ok := getJSONBool(raw, "required"); ok {
		attr.Required = value
	}
	if value, ok := getJSONBool(raw, "foreignKey", "foreign_key", "foreignkey", "fk"); ok {
		attr.ForeignKey = value
	}
	attr.MaxLength = getJSONInt(raw, "maxLength", "max_length", "maxlen", "maxLen")

	if attr.Type == "" || attr.Name == "" {
		return Attribute{}, errors.New("type and name are required")
	}
	if !isCSharpIdentifier(attr.Name) {
		return Attribute{}, fmt.Errorf("%q is not a valid C# property name", attr.Name)
	}
	if attr.MaxLength < 0 {
		return Attribute{}, errors.New("maxLength must be greater than zero")
	}

	return attr, nil
}

func getJSONInt(raw map[string]json.RawMessage, names ...string) int {
	for _, name := range names {
		value, ok := raw[name]
		if !ok {
			continue
		}
		var parsed int
		if err := json.Unmarshal(value, &parsed); err == nil {
			return parsed
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func ParseAttribute(spec string) (Attribute, error) {
	parts := strings.Fields(strings.TrimSpace(spec))
	if len(parts) < 2 {
		return Attribute{}, errors.New("expected: <type> <name> [required] [maxlen:N] [foreignkey]")
	}

	attr := Attribute{
		Type: strings.TrimSpace(parts[0]),
		Name: toPascal(parts[1]),
	}

	if attr.Type == "" || attr.Name == "" {
		return Attribute{}, errors.New("type and name are required")
	}
	if !isCSharpIdentifier(attr.Name) {
		return Attribute{}, fmt.Errorf("%q is not a valid C# property name", attr.Name)
	}

	for _, mod := range parts[2:] {
		normalized := strings.ToLower(strings.TrimSpace(mod))
		switch {
		case normalized == "required":
			attr.Required = true
		case normalized == "foreignkey" || normalized == "foreign" || normalized == "fk":
			attr.ForeignKey = true
		case strings.HasPrefix(normalized, "maxlen:") || strings.HasPrefix(normalized, "maxlength:") || strings.HasPrefix(normalized, "max:"):
			_, value, _ := strings.Cut(normalized, ":")
			maxLength, err := strconv.Atoi(value)
			if err != nil || maxLength <= 0 {
				return Attribute{}, fmt.Errorf("invalid max length modifier %q", mod)
			}
			attr.MaxLength = maxLength
		default:
			return Attribute{}, fmt.Errorf("unknown modifier %q", mod)
		}
	}

	return attr, nil
}

func BuildPlan(opts Options) (*Plan, error) {
	if strings.TrimSpace(opts.Entity) == "" {
		return nil, errors.New("--entity is required")
	}

	opts.Entity = toPascal(opts.Entity)
	if !isCSharpIdentifier(opts.Entity) {
		return nil, fmt.Errorf("--entity %q is not a valid C# class name", opts.Entity)
	}

	if strings.TrimSpace(opts.EntityType) == "" {
		opts.EntityType = "FullAuditedAggregateRoot<Guid>"
	}

	if strings.EqualFold(opts.EntityType, "ValueObject") {
		return nil, errors.New("CRUD generation requires an entity or aggregate root; ValueObject is not supported for CRUD files")
	}

	if opts.KeyType == "" {
		opts.KeyType = inferKeyType(opts.EntityType)
	}
	if opts.KeyType == "" {
		return nil, errors.New("--key is required when --entity-type does not include a generic key type")
	}
	opts.Mapping = normalizeMappingStyle(opts.Mapping)
	if opts.Mapping == "" {
		return nil, errors.New("--mapping must be auto, automapper, or mapperly")
	}
	files, err := normalizeFileTypes(opts.Files)
	if err != nil {
		return nil, err
	}
	opts.Files = files

	info, err := DetectApp(opts.Root, opts.Module)
	if err != nil {
		return nil, err
	}
	if err := requireSelectedLayerDirs(info, opts); err != nil {
		return nil, err
	}
	if opts.Mapping != mappingAuto {
		info.MappingStyle = opts.Mapping
		info.MappingStyleSource = "explicit --mapping/config value"
	}
	opts.Root = info.Root
	opts.Module = info.Module

	plan := &Plan{
		Info:    info,
		Options: opts,
	}

	if err := addCRUDChanges(plan); err != nil {
		return nil, err
	}
	sort.SliceStable(plan.Changes, func(i, j int) bool {
		return plan.Changes[i].Path < plan.Changes[j].Path
	})

	return plan, nil
}

func DetectApp(root string, requestedModule string) (AppInfo, error) {
	startDir, err := absoluteDirectory(root)
	if err != nil {
		return AppInfo{}, err
	}

	absRoot, err := resolveABPRoot(root)
	if err != nil {
		return AppInfo{}, err
	}

	info := AppInfo{
		Root:      absRoot,
		LayerDirs: map[string]string{},
	}

	info.SolutionPath = findFirstFile(absRoot, ".sln")

	modules, srcDir := detectModules(absRoot)
	info.DetectedModuleNames = sortedKeys(modules)
	if requestedModule == "" {
		requestedModule = inferModuleFromPath(startDir, absRoot, modules)
	}
	if requestedModule != "" {
		if _, ok := modules[requestedModule]; !ok {
			return AppInfo{}, fmt.Errorf("module %q was not found under %s. Detected modules: %s", requestedModule, filepath.Join(absRoot, "src"), strings.Join(info.DetectedModuleNames, ", "))
		}
		info.Module = requestedModule
	} else {
		module, err := chooseModule(modules, info.SolutionPath)
		if err != nil {
			return AppInfo{}, err
		}
		info.Module = module
	}

	for _, layer := range layerNames {
		if path, ok := modules[info.Module][layer]; ok {
			info.LayerDirs[layer] = path
		} else {
			info.LayerDirs[layer] = filepath.Join(srcDir, info.Module+"."+layer)
		}
	}

	info.ABPVersion, info.ABPVersionSource = detectABPVersion(absRoot)
	info.ABPMajor = parseMajor(info.ABPVersion)

	info.DbContextFile, info.DbContextName = detectDbContext(info.LayerDirs[layerEntityFrameworkCore], info.Module)
	if info.DbContextName != "" {
		info.DbContextPrefix = strings.TrimSuffix(info.DbContextName, "DbContext")
	}
	if info.DbContextPrefix == "" {
		info.DbContextPrefix = lastNamespaceSegment(info.Module)
	}
	if info.DbContextName == "" {
		info.DbContextName = info.DbContextPrefix + "DbContext"
	}
	info.IDbContextFile = detectIDbContext(info.LayerDirs[layerEntityFrameworkCore], info.DbContextPrefix)
	info.DbPropertiesName, info.DbPropertiesNamespace = detectDbProperties(absRoot, info.DbContextPrefix)
	if info.DbPropertiesName == "" {
		info.DbPropertiesName = info.DbContextPrefix + "DbProperties"
	}
	if info.DbPropertiesNamespace == "" {
		info.DbPropertiesNamespace = info.Module + ".EntityFrameworkCore"
	}
	info.ResourceType, info.ResourceNamespace = detectLocalizationResource(info.LayerDirs[layerDomainShared], info.Module)
	if info.ResourceType == "" {
		info.ResourceType = lastNamespaceSegment(info.Module) + "Resource"
	}
	if info.ResourceNamespace == "" {
		info.ResourceNamespace = info.Module + ".Localization"
	}
	info.MappingStyle, info.MappingStyleSource = detectMappingStyle(absRoot, info.ABPMajor)

	return info, nil
}

func resolveABPRoot(root string) (string, error) {
	start, err := absoluteDirectory(root)
	if err != nil {
		return "", err
	}

	for dir := start; ; dir = filepath.Dir(dir) {
		if hasDetectedModules(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return start, nil
}

func absoluteDirectory(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if stat, statErr := os.Stat(absPath); statErr == nil && !stat.IsDir() {
		return filepath.Dir(absPath), nil
	}
	return absPath, nil
}

func hasDetectedModules(root string) bool {
	modules, _ := detectModules(root)
	return len(modules) > 0
}

func inferModuleFromPath(startDir string, root string, modules map[string]map[string]string) string {
	srcDir := filepath.Join(root, "src")
	rel, err := filepath.Rel(srcDir, startDir)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return ""
	}

	firstSegment := strings.Split(rel, string(os.PathSeparator))[0]
	for _, layer := range layerNames {
		suffix := "." + layer
		if strings.HasSuffix(firstSegment, suffix) {
			module := strings.TrimSuffix(firstSegment, suffix)
			if _, ok := modules[module]; ok {
				return module
			}
			return ""
		}
	}
	return ""
}

func PrintPlan(w io.Writer, plan *Plan) {
	info := plan.Info
	opts := plan.Options

	fmt.Fprintln(w, "ABP CRUD Generator")
	fmt.Fprintln(w, "==================")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Detected app")
	fmt.Fprintf(w, "  Root       : %s\n", info.Root)
	if info.SolutionPath != "" {
		fmt.Fprintf(w, "  Solution   : %s\n", info.SolutionPath)
	}
	fmt.Fprintf(w, "  Module     : %s\n", info.Module)
	if info.ABPVersion != "" {
		fmt.Fprintf(w, "  ABP version: %s", info.ABPVersion)
		if info.ABPVersionSource != "" {
			fmt.Fprintf(w, " (%s)", info.ABPVersionSource)
		}
		fmt.Fprintln(w)
	} else {
		fmt.Fprintln(w, "  ABP version: not found")
	}
	fmt.Fprintf(w, "  Mapping    : %s", info.MappingStyle)
	if info.MappingStyleSource != "" {
		fmt.Fprintf(w, " (%s)", info.MappingStyleSource)
	}
	fmt.Fprintln(w)
	if opts.TemplateDir != "" {
		fmt.Fprintf(w, "  Templates  : %s\n", opts.TemplateDir)
	} else {
		fmt.Fprintln(w, "  Templates  : built-in")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "CRUD target")
	fmt.Fprintf(w, "  Entity     : %s\n", opts.Entity)
	fmt.Fprintf(w, "  Entity type: %s\n", opts.EntityType)
	fmt.Fprintf(w, "  Key type   : %s\n", opts.KeyType)
	fmt.Fprintf(w, "  Files      : %s\n", strings.Join(opts.Files, ", "))
	if len(opts.Attrs) == 0 {
		fmt.Fprintln(w, "  Attributes : none")
	} else {
		fmt.Fprintln(w, "  Attributes :")
		for _, attr := range opts.Attrs {
			fmt.Fprintf(w, "    - %s %s", attr.Type, attr.Name)
			if attr.Required {
				fmt.Fprint(w, " required")
			}
			if attr.MaxLength > 0 {
				fmt.Fprintf(w, " maxlen:%d", attr.MaxLength)
			}
			if attr.ForeignKey {
				fmt.Fprint(w, " foreignkey")
			}
			fmt.Fprintln(w)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Compatibility notes")
	for _, note := range compatibilityNotes(info) {
		fmt.Fprintf(w, "  - %s\n", note)
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Planned file changes (%d)\n", len(plan.Changes))
	summary := summarizeChanges(plan.Changes)
	fmt.Fprintf(w, "  Summary    : create=%d update=%d overwrite=%d skip=%d unchanged=%d\n",
		summary["create"], summary["update"], summary["overwrite"], summary["skip-existing"], summary["unchanged"])
	for _, change := range plan.Changes {
		fmt.Fprintf(w, "  %-13s %-30s %s\n", "["+change.Action+"]", change.Type, change.Path)
	}
}

func summarizeChanges(changes []FileChange) map[string]int {
	summary := map[string]int{}
	for _, change := range changes {
		summary[change.Action]++
	}
	return summary
}

func AskForApproval(stdin io.Reader, stdout io.Writer) bool {
	fmt.Fprint(stdout, "\nGenerate all listed changes? Type yes to continue: ")
	reader := bufio.NewReader(stdin)
	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(answer), "yes")
}

func ApplyPlan(plan *Plan) error {
	for _, change := range plan.Changes {
		switch change.Action {
		case "create", "update", "overwrite":
			if err := os.MkdirAll(filepath.Dir(change.Path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(change.Path, change.Content, 0o644); err != nil {
				return err
			}
		case "error":
			return fmt.Errorf("cannot apply plan because %s failed to plan: %s", change.Type, change.Path)
		}
	}
	return nil
}

func addCRUDChanges(plan *Plan) error {
	info := plan.Info
	opts := plan.Options
	entity := opts.Entity
	module := info.Module

	domainShared := info.LayerDirs[layerDomainShared]
	domain := info.LayerDirs[layerDomain]
	contracts := info.LayerDirs[layerApplicationContracts]
	application := info.LayerDirs[layerApplication]
	httpAPI := info.LayerDirs[layerHttpAPI]
	efCore := info.LayerDirs[layerEntityFrameworkCore]

	if err := addGeneratedTemplateFile(plan, fileTypeConstants, filepath.Join(domainShared, "Constants", entity+"Constants.cs"), constantsTemplate); err != nil {
		return err
	}
	if err := addGeneratedTemplateFile(plan, fileTypeEventTypes, filepath.Join(domainShared, "Events", entity+"EtoTypes.cs"), eventTypesTemplate); err != nil {
		return err
	}
	if err := addGeneratedTemplateFile(plan, fileTypeEto, filepath.Join(domainShared, "Events", entity+"Eto.cs"), etoTemplate); err != nil {
		return err
	}

	if err := addGeneratedTemplateFile(plan, fileTypeEntity, filepath.Join(domain, "Entities", entity+".cs"), entityTemplate); err != nil {
		return err
	}
	if err := addGeneratedTemplateFile(plan, fileTypeRepositoryInterface, filepath.Join(domain, "Repositories", "I"+entity+"Repository.cs"), repositoryInterfaceTemplate); err != nil {
		return err
	}
	if err := addGeneratedTemplateFile(plan, fileTypeDataSeeder, filepath.Join(domain, "Data", entity+"DataSeeder.cs"), dataSeederTemplate); err != nil {
		return err
	}

	if err := addGeneratedTemplateFile(plan, fileTypeCreateDto, filepath.Join(contracts, entity, "Create"+entity+"Dto.cs"), createDtoTemplate); err != nil {
		return err
	}
	if err := addGeneratedTemplateFile(plan, fileTypeUpdateDto, filepath.Join(contracts, entity, "Update"+entity+"Dto.cs"), updateDtoTemplate); err != nil {
		return err
	}
	if err := addGeneratedTemplateFile(plan, fileTypeEntityDto, filepath.Join(contracts, entity, entity+"Dto.cs"), readDtoTemplate); err != nil {
		return err
	}
	if err := addGeneratedTemplateFile(plan, fileTypeAppServiceInterface, filepath.Join(contracts, "Services", "I"+entity+"AppService.cs"), appServiceInterfaceTemplate); err != nil {
		return err
	}

	if err := addUpsertTemplateFile(plan, fileTypePermissions, filepath.Join(contracts, "Permissions", permissionClassName(info)+".cs"), permissionsTemplate, func(existing string) (string, bool, error) {
		return updatePermissions(existing, info, opts)
	}); err != nil {
		return err
	}
	if err := addUpsertTemplateFile(plan, fileTypePermissionDefinitionProvider, filepath.Join(contracts, "Permissions", permissionDefinitionProviderClassName(info)+".cs"), permissionDefinitionProviderTemplate, func(existing string) (string, bool, error) {
		return updatePermissionDefinitionProvider(existing, info, opts)
	}); err != nil {
		return err
	}

	if err := addGeneratedTemplateFile(plan, fileTypeAppService, filepath.Join(application, "Services", entity+"AppService.cs"), appServiceTemplate); err != nil {
		return err
	}
	if info.MappingStyle == mappingMapperly {
		if err := addGeneratedTemplateFile(plan, fileTypeMapping, filepath.Join(application, "Mapperly", entity+"Mappers.cs"), mapperlyMappersTemplate); err != nil {
			return err
		}
	} else {
		if err := addGeneratedTemplateFile(plan, fileTypeMapping, filepath.Join(application, "AutoMapper", entity+"Profile.cs"), autoMapperProfileTemplate); err != nil {
			return err
		}
	}

	if err := addGeneratedTemplateFile(plan, fileTypeController, filepath.Join(httpAPI, "Controllers", entity+"Controller.cs"), controllerTemplate); err != nil {
		return err
	}

	configurationPath := filepath.Join(efCore, "EntityFrameworkCore", "Configurations", entity+"Configuration.cs")
	if err := addGeneratedTemplateFile(plan, fileTypeEFConfiguration, configurationPath, efConfigurationTemplate); err != nil {
		return err
	}
	if err := addGeneratedTemplateFile(plan, fileTypeEFRepository, filepath.Join(efCore, "EntityFrameworkCore", "Repositories", "EfCore"+entity+"Repository.cs"), efRepositoryTemplate); err != nil {
		return err
	}

	if includesFileType(opts, fileTypeDbContext) && info.DbContextFile == "" {
		return fmt.Errorf("DbContext file was not detected under %s; create it first or omit --file %s", efCore, fileTypeDbContext)
	}
	if info.DbContextFile != "" {
		applyConfiguration := includesFileType(opts, fileTypeEFConfiguration) || fileExists(configurationPath)
		if err := addUpsertFile(plan, fileTypeDbContext, info.DbContextFile, "", func(existing string) (string, bool, error) {
			return updateDbContext(existing, info, opts, applyConfiguration)
		}); err != nil {
			return err
		}
	}
	if info.IDbContextFile != "" {
		if err := addUpsertTemplateFile(plan, fileTypeIDbContext, info.IDbContextFile, iDbContextTemplate, func(existing string) (string, bool, error) {
			return updateIDbContext(existing, info, opts)
		}); err != nil {
			return err
		}
	} else {
		path := filepath.Join(efCore, "EntityFrameworkCore", "I"+info.DbContextPrefix+"DbContext.cs")
		if err := addGeneratedTemplateFile(plan, fileTypeIDbContext, path, iDbContextTemplate); err != nil {
			return err
		}
	}

	for _, item := range []struct {
		culture  string
		fileType string
	}{
		{culture: "en", fileType: fileTypeLocalizationEN},
		{culture: "ar", fileType: fileTypeLocalizationAR},
	} {
		path := filepath.Join(domainShared, "Localization", lastNamespaceSegment(module), item.culture+".json")
		if err := addUpsertLocalizedTemplateFile(plan, item.fileType, path, item.culture, func(existing string) (string, bool, error) {
			return updateLocalization(existing, item.culture, info, opts)
		}); err != nil {
			return err
		}
	}
	return nil
}

func addGeneratedTemplateFile(plan *Plan, fileType string, path string, render func(AppInfo, Options) (string, error)) error {
	if !includesFileType(plan.Options, fileType) {
		return nil
	}

	content, err := render(plan.Info, plan.Options)
	if err != nil {
		return fmt.Errorf("render %s template: %w", fileType, err)
	}
	addGeneratedFile(plan, fileType, path, content)
	return nil
}

func addUpsertTemplateFile(plan *Plan, fileType string, path string, render func(AppInfo, Options) (string, error), update func(existing string) (string, bool, error)) error {
	if !includesFileType(plan.Options, fileType) {
		return nil
	}

	createContent, err := render(plan.Info, plan.Options)
	if err != nil {
		return fmt.Errorf("render %s template: %w", fileType, err)
	}
	return addUpsertFile(plan, fileType, path, createContent, update)
}

func addUpsertLocalizedTemplateFile(plan *Plan, fileType string, path string, culture string, update func(existing string) (string, bool, error)) error {
	if !includesFileType(plan.Options, fileType) {
		return nil
	}

	createContent, err := localizationTemplate(plan.Info, plan.Options, culture)
	if err != nil {
		return fmt.Errorf("render %s template: %w", fileType, err)
	}
	return addUpsertFile(plan, fileType, path, createContent, update)
}

func addGeneratedFile(plan *Plan, fileType string, path string, content string) {
	if !includesFileType(plan.Options, fileType) {
		return
	}
	absPath, _ := filepath.Abs(path)
	action := "create"
	if existing, err := os.ReadFile(absPath); err == nil {
		if string(existing) == content {
			action = "unchanged"
		} else if plan.Options.Overwrite {
			action = "overwrite"
		} else {
			action = "skip-existing"
			content = string(existing)
		}
	}
	plan.Changes = append(plan.Changes, FileChange{
		Type:    fileType,
		Path:    absPath,
		Action:  action,
		Content: []byte(content),
	})
}

func addUpsertFile(plan *Plan, fileType string, path string, createContent string, update func(existing string) (string, bool, error)) error {
	if !includesFileType(plan.Options, fileType) {
		return nil
	}
	absPath, _ := filepath.Abs(path)
	if existing, err := os.ReadFile(absPath); err == nil {
		next, changed, updateErr := update(string(existing))
		if updateErr != nil {
			return fmt.Errorf("failed to update %s (%s): %w", absPath, fileType, updateErr)
		}
		action := "unchanged"
		if changed && next != string(existing) {
			action = "update"
		}
		plan.Changes = append(plan.Changes, FileChange{
			Type:    fileType,
			Path:    absPath,
			Action:  action,
			Content: []byte(next),
		})
		return nil
	}

	plan.Changes = append(plan.Changes, FileChange{
		Type:    fileType,
		Path:    absPath,
		Action:  "create",
		Content: []byte(createContent),
	})
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func requireSelectedLayerDirs(info AppInfo, opts Options) error {
	selectedLayers := map[string]bool{}
	add := func(layer string) {
		selectedLayers[layer] = true
	}

	for _, fileType := range opts.Files {
		switch fileType {
		case fileTypeConstants, fileTypeEventTypes, fileTypeEto, fileTypeLocalizationEN, fileTypeLocalizationAR:
			add(layerDomainShared)
		case fileTypeEntity, fileTypeRepositoryInterface, fileTypeDataSeeder:
			add(layerDomain)
		case fileTypeCreateDto, fileTypeUpdateDto, fileTypeEntityDto, fileTypeAppServiceInterface, fileTypePermissions, fileTypePermissionDefinitionProvider:
			add(layerApplicationContracts)
		case fileTypeAppService, fileTypeMapping:
			add(layerApplication)
		case fileTypeController:
			add(layerHttpAPI)
		case fileTypeEFConfiguration, fileTypeEFRepository, fileTypeDbContext, fileTypeIDbContext:
			add(layerEntityFrameworkCore)
		}
	}

	var missing []string
	for _, layer := range layerNames {
		if !selectedLayers[layer] {
			continue
		}
		if _, err := os.Stat(info.LayerDirs[layer]); err != nil {
			missing = append(missing, info.LayerDirs[layer])
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("selected ABP layer projects were not found. Adjust --root/--module or choose fewer --file values. Missing:\n  %s", strings.Join(missing, "\n  "))
	}
	return nil
}

func detectModules(root string) (map[string]map[string]string, string) {
	srcDir := filepath.Join(root, "src")
	modules := map[string]map[string]string{}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return modules, srcDir
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		for _, layer := range layerNames {
			suffix := "." + layer
			if strings.HasSuffix(name, suffix) {
				module := strings.TrimSuffix(name, suffix)
				if modules[module] == nil {
					modules[module] = map[string]string{}
				}
				modules[module][layer] = filepath.Join(srcDir, name)
				break
			}
		}
	}

	return modules, srcDir
}

func chooseModule(modules map[string]map[string]string, solutionPath string) (string, error) {
	if len(modules) == 0 {
		return "", errors.New("no ABP module layer projects were detected under src")
	}

	if solutionPath != "" {
		base := strings.TrimSuffix(filepath.Base(solutionPath), filepath.Ext(solutionPath))
		if _, ok := modules[base]; ok {
			return base, nil
		}
	}

	var candidates []string
	for module, layers := range modules {
		if hasRequiredLayers(layers) {
			candidates = append(candidates, module)
		}
	}
	sort.Strings(candidates)
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf("multiple ABP modules were detected (%s). Pass --module to choose one", strings.Join(candidates, ", "))
	}

	all := sortedKeys(modules)
	return all[0], nil
}

func hasRequiredLayers(layers map[string]string) bool {
	for _, layer := range []string{layerDomainShared, layerDomain, layerApplicationContracts, layerApplication, layerEntityFrameworkCore} {
		if layers[layer] == "" {
			return false
		}
	}
	return true
}

func detectABPVersion(root string) (string, string) {
	props := map[string]string{}
	var hits []struct {
		version string
		source  string
	}

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".csproj" && ext != ".props" && ext != ".targets" {
			return nil
		}
		contentBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := string(contentBytes)
		for name, value := range findMSBuildProperties(content) {
			props[name] = value
		}
		return nil
	})

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".csproj" && ext != ".props" && ext != ".targets" {
			return nil
		}
		contentBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := string(contentBytes)
		for _, rawVersion := range findVoloAbpVersions(content) {
			version := resolveMSBuildValue(rawVersion, props)
			if version == "" || strings.Contains(version, "$(") {
				continue
			}
			hits = append(hits, struct {
				version string
				source  string
			}{version: version, source: path})
		}
		return nil
	})

	if len(hits) == 0 {
		if version := props["AbpVersion"]; version != "" {
			return version, "MSBuild property AbpVersion"
		}
		return "", ""
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].version == hits[j].version {
			return hits[i].source < hits[j].source
		}
		return compareVersion(hits[i].version, hits[j].version) > 0
	})

	return hits[0].version, hits[0].source
}

func detectMappingStyle(root string, abpMajor int) (string, string) {
	hits := scanMappingMarkers(root)
	if hits[mappingMapperly] > 0 {
		return mappingMapperly, "found Mapperly package/module markers"
	}
	if hits[mappingAutoMapper] > 0 {
		return mappingAutoMapper, "found AutoMapper package/module markers"
	}
	if abpMajor >= 10 {
		return mappingMapperly, "ABP 10+ default"
	}
	return mappingAutoMapper, "ABP 8/9 default"
}

func scanMappingMarkers(root string) map[string]int {
	hits := map[string]int{
		mappingAutoMapper: 0,
		mappingMapperly:   0,
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".cs" && ext != ".csproj" && ext != ".props" && ext != ".targets" {
			return nil
		}
		contentBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := string(contentBytes)
		if strings.Contains(content, "Volo.Abp.Mapperly") ||
			strings.Contains(content, "AbpMapperlyModule") ||
			strings.Contains(content, "AddMapperlyObjectMapper") ||
			strings.Contains(content, "Riok.Mapperly") ||
			strings.Contains(content, "MapperBase<") ||
			strings.Contains(content, "TwoWayMapperBase<") {
			hits[mappingMapperly]++
		}
		if strings.Contains(content, "Volo.Abp.AutoMapper") ||
			strings.Contains(content, "AbpAutoMapperModule") ||
			strings.Contains(content, "AddAutoMapperObjectMapper") ||
			strings.Contains(content, "AutoMapper") ||
			strings.Contains(content, ": Profile") {
			hits[mappingAutoMapper]++
		}
		return nil
	})
	return hits
}

func normalizeMappingStyle(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", mappingAuto:
		return mappingAuto
	case "auto-mapper", "auto_mapper", mappingAutoMapper:
		return mappingAutoMapper
	case "mapper-ly", "mapper_ly", mappingMapperly:
		return mappingMapperly
	default:
		return ""
	}
}

func compatibilityNotes(info AppInfo) []string {
	switch {
	case info.ABPMajor >= 10:
		return []string{
			"ABP 10 targets .NET 10 and ABP modules moved to Mapperly; generated mapper files follow the detected mapping style.",
			"Create an EF Core migration after generation or upgrade if the DbContext/model changed.",
		}
	case info.ABPMajor == 9:
		return []string{
			"ABP 9 targets .NET 9 and still works with AutoMapper-style CRUD mapping in existing templates.",
			"Host static asset changes such as MapAbpStaticAssets are upgrade concerns, not CRUD file generation changes.",
		}
	case info.ABPMajor == 8:
		return []string{
			"ABP 8 targets .NET 8 and uses the same core CRUD surfaces generated here.",
			"Read-only repositories introduced in ABP 8 are query-only guidance; CRUD services still use IRepository.",
		}
	default:
		return []string{
			"ABP version could not be mapped to 8, 9, or 10; generation uses detected project conventions where possible.",
		}
	}
}

func findMSBuildProperties(content string) map[string]string {
	props := map[string]string{}
	re := regexp.MustCompile(`<([A-Za-z_][A-Za-z0-9_.-]*)>\s*([^<]+)\s*</([A-Za-z_][A-Za-z0-9_.-]*)>`)
	for _, match := range re.FindAllStringSubmatch(content, -1) {
		if match[1] != match[3] {
			continue
		}
		props[match[1]] = strings.TrimSpace(match[2])
	}
	return props
}

func findVoloAbpVersions(content string) []string {
	var versions []string
	inline := regexp.MustCompile(`<(?:PackageReference|PackageVersion)\b[^>]*Include\s*=\s*"Volo\.Abp[^"]*"[^>]*Version\s*=\s*"([^"]+)"`)
	for _, match := range inline.FindAllStringSubmatch(content, -1) {
		versions = append(versions, strings.TrimSpace(match[1]))
	}
	reversed := regexp.MustCompile(`<(?:PackageReference|PackageVersion)\b[^>]*Version\s*=\s*"([^"]+)"[^>]*Include\s*=\s*"Volo\.Abp[^"]*"`)
	for _, match := range reversed.FindAllStringSubmatch(content, -1) {
		versions = append(versions, strings.TrimSpace(match[1]))
	}
	nestedBlock := regexp.MustCompile(`(?s)<PackageReference\b[^>]*Include\s*=\s*"Volo\.Abp[^"]*"[^>]*>(.*?)</PackageReference>`)
	nestedVersion := regexp.MustCompile(`<Version>\s*([^<]+)\s*</Version>`)
	for _, block := range nestedBlock.FindAllStringSubmatch(content, -1) {
		if match := nestedVersion.FindStringSubmatch(block[1]); match != nil {
			versions = append(versions, strings.TrimSpace(match[1]))
		}
	}
	return versions
}

func resolveMSBuildValue(value string, props map[string]string) string {
	re := regexp.MustCompile(`^\$\(([A-Za-z_][A-Za-z0-9_.-]*)\)$`)
	match := re.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(props[match[1]])
}

func detectDbContext(efCoreDir string, module string) (string, string) {
	var files []string
	_ = filepath.WalkDir(efCoreDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		name := filepath.Base(path)
		if strings.HasSuffix(name, "DbContext.cs") && !strings.HasPrefix(name, "I") && !strings.Contains(name, "Factory") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)

	preferred := lastNamespaceSegment(module) + "DbContext"
	for _, path := range files {
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		className := firstClassName(string(contentBytes), "DbContext")
		if className == preferred {
			return path, className
		}
	}
	for _, path := range files {
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		className := firstClassName(string(contentBytes), "DbContext")
		if className != "" {
			return path, className
		}
	}
	return "", ""
}

func detectIDbContext(efCoreDir string, prefix string) string {
	var files []string
	_ = filepath.WalkDir(efCoreDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		name := filepath.Base(path)
		if strings.HasPrefix(name, "I") && strings.HasSuffix(name, "DbContext.cs") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	preferred := "I" + prefix + "DbContext.cs"
	for _, path := range files {
		if filepath.Base(path) == preferred {
			return path
		}
	}
	if len(files) > 0 {
		return files[0]
	}
	return ""
}

func detectDbProperties(root string, prefix string) (string, string) {
	var fallbackName string
	var fallbackNamespace string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(filepath.Base(path), "DbProperties.cs") {
			return nil
		}
		contentBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := string(contentBytes)
		className := firstClassName(content, "DbProperties")
		if className == "" {
			return nil
		}
		namespaceName := firstNamespace(content)
		if fallbackName == "" {
			fallbackName = className
			fallbackNamespace = namespaceName
		}
		if className == prefix+"DbProperties" {
			fallbackName = className
			fallbackNamespace = namespaceName
			return filepath.SkipAll
		}
		return nil
	})
	return fallbackName, fallbackNamespace
}

func detectLocalizationResource(domainSharedDir string, module string) (string, string) {
	var resourceType string
	var resourceNamespace string
	preferred := lastNamespaceSegment(module) + "Resource"

	_ = filepath.WalkDir(domainSharedDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(filepath.Base(path), "Resource.cs") {
			return nil
		}
		contentBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := string(contentBytes)
		className := firstClassName(content, "Resource")
		namespaceName := firstNamespace(content)
		if className == "" {
			return nil
		}
		if resourceType == "" || className == preferred {
			resourceType = className
			resourceNamespace = namespaceName
		}
		if className == preferred {
			return filepath.SkipAll
		}
		return nil
	})

	return resourceType, resourceNamespace
}

func firstClassName(content string, suffix string) string {
	re := regexp.MustCompile(`\bclass\s+([A-Za-z_][A-Za-z0-9_]*` + regexp.QuoteMeta(suffix) + `)\b`)
	match := re.FindStringSubmatch(content)
	if match == nil {
		return ""
	}
	return match[1]
}

func firstNamespace(content string) string {
	fileScoped := regexp.MustCompile(`(?m)^\s*namespace\s+([A-Za-z_][A-Za-z0-9_.]*)\s*;`)
	if match := fileScoped.FindStringSubmatch(content); match != nil {
		return match[1]
	}
	block := regexp.MustCompile(`(?m)^\s*namespace\s+([A-Za-z_][A-Za-z0-9_.]*)\s*(?:\{|$)`)
	if match := block.FindStringSubmatch(content); match != nil {
		return match[1]
	}
	return ""
}

func findFirstFile(root string, extension string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), extension) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".vs", ".idea", "bin", "obj", "node_modules", "packages":
		return true
	default:
		return false
	}
}

func updatePermissions(existing string, info AppInfo, opts Options) (string, bool, error) {
	if strings.Contains(existing, opts.Entity+"Management") {
		return existing, false, nil
	}

	insert := fmt.Sprintf(`        public static class %sManagement
        {
            public const string Default = GroupName + ".%s";
            public const string Create = Default + ".Create";
            public const string Update = Default + ".Update";
            public const string Delete = Default + ".Delete";
        }

`, opts.Entity, opts.Entity)

	if idx := strings.Index(existing, "public static string[] GetAll()"); idx >= 0 {
		return existing[:idx] + insert + existing[idx:], true, nil
	}

	idx := strings.LastIndex(existing, "}")
	if idx < 0 {
		return existing + "\n" + insert, true, nil
	}
	return existing[:idx] + insert + existing[idx:], true, nil
}

func updatePermissionDefinitionProvider(existing string, info AppInfo, opts Options) (string, bool, error) {
	permissionsClass := permissionClassName(info)
	if strings.Contains(existing, permissionsClass+"."+opts.Entity+"Management.Default") {
		return existing, false, nil
	}

	groupVar := detectPermissionGroupVariable(existing)
	if groupVar == "" {
		groupVar = "group"
	}

	entityVar := lowerFirst(opts.Entity) + "Permission"
	insert := fmt.Sprintf(`
            var %s = %s.AddPermission(%s.%sManagement.Default, L("Permission:%s"));
            %s.AddChild(%s.%sManagement.Create, L("Permission:%s.Create"));
            %s.AddChild(%s.%sManagement.Update, L("Permission:%s.Update"));
            %s.AddChild(%s.%sManagement.Delete, L("Permission:%s.Delete"));
`, entityVar, groupVar, permissionsClass, opts.Entity, opts.Entity,
		entityVar, permissionsClass, opts.Entity, opts.Entity,
		entityVar, permissionsClass, opts.Entity, opts.Entity,
		entityVar, permissionsClass, opts.Entity, opts.Entity)

	next, ok := insertBeforeMethodClosingBrace(existing, "Define", insert)
	if ok {
		return next, true, nil
	}

	idx := strings.LastIndex(existing, "}")
	if idx < 0 {
		return existing + insert, true, nil
	}
	return existing[:idx] + insert + existing[idx:], true, nil
}

func detectPermissionGroupVariable(content string) string {
	re := regexp.MustCompile(`var\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*context\.AddGroup\(`)
	match := re.FindStringSubmatch(content)
	if match == nil {
		return ""
	}
	return match[1]
}

func updateDbContext(existing string, info AppInfo, opts Options, applyConfiguration bool) (string, bool, error) {
	next := existing
	next = ensureUsing(next, "Microsoft.EntityFrameworkCore")
	next = ensureUsing(next, info.Module+".Domain.Entities")
	if applyConfiguration {
		next = ensureUsing(next, info.Module+".EntityFrameworkCore.Configurations")
	}

	changed := next != existing
	plural := pluralize(opts.Entity)
	memberIndent := typeMemberIndent(next)
	dbSet := fmt.Sprintf("%spublic DbSet<%s> %s { get; set; }\n", memberIndent, opts.Entity, plural)
	if !strings.Contains(next, "DbSet<"+opts.Entity+">") {
		if markerIdx := strings.Index(next, "/* Add DbSet properties"); markerIdx >= 0 {
			lineEnd := strings.Index(next[markerIdx:], "\n")
			if lineEnd >= 0 {
				insertAt := markerIdx + lineEnd + 1
				next = next[:insertAt] + dbSet + next[insertAt:]
			} else {
				next += "\n" + dbSet
			}
		} else {
			inserted, ok := insertAfterClassOpeningBrace(next, info.DbContextName, dbSet)
			if ok {
				next = inserted
			} else {
				next += "\n" + dbSet
			}
		}
		changed = true
	}

	configLine := fmt.Sprintf("%sbuilder.ApplyConfiguration(new %sConfiguration());\n", memberIndent+"    ", opts.Entity)
	if applyConfiguration && !strings.Contains(next, "new "+opts.Entity+"Configuration()") {
		inserted, ok := insertOnModelCreatingLine(next, configLine)
		if ok {
			next = inserted
			changed = true
		}
	}

	return next, changed, nil
}

func updateIDbContext(existing string, info AppInfo, opts Options) (string, bool, error) {
	next := existing
	next = ensureUsing(next, "Microsoft.EntityFrameworkCore")
	next = ensureUsing(next, info.Module+".Domain.Entities")
	if info.DbPropertiesNamespace != "" {
		next = ensureUsing(next, info.DbPropertiesNamespace)
	}
	changed := next != existing

	if !strings.Contains(next, "DbSet<"+opts.Entity+">") {
		dbSet := fmt.Sprintf("%sDbSet<%s> %s { get; }\n", typeMemberIndent(next), opts.Entity, pluralize(opts.Entity))
		interfaceName := "I" + info.DbContextPrefix + "DbContext"
		inserted, ok := insertAfterClassOpeningBrace(next, interfaceName, dbSet)
		if ok {
			next = inserted
		} else {
			idx := strings.LastIndex(next, "}")
			if idx >= 0 {
				next = next[:idx] + dbSet + next[idx:]
			} else {
				next += "\n" + dbSet
			}
		}
		changed = true
	}

	return next, changed, nil
}

func updateLocalization(existing string, culture string, info AppInfo, opts Options) (string, bool, error) {
	if strings.TrimSpace(existing) == "" {
		created, err := localizationTemplate(info, opts, culture)
		if err != nil {
			return "", false, err
		}
		existing = created
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(existing), &doc); err != nil {
		return "", false, err
	}
	if doc == nil {
		doc = map[string]any{}
	}
	if _, ok := doc["culture"]; !ok {
		doc["culture"] = culture
	}

	texts, ok := doc["texts"].(map[string]any)
	if !ok || texts == nil {
		texts = map[string]any{}
	}

	changed := false
	add := func(key, value string) {
		if _, exists := texts[key]; !exists {
			texts[key] = value
			changed = true
		}
	}

	add(opts.Entity, opts.Entity)
	add(opts.Entity+"Management", opts.Entity+" Management")
	add("Create"+opts.Entity, "Create "+opts.Entity)
	add("Edit"+opts.Entity, "Edit "+opts.Entity)
	add("Delete"+opts.Entity, "Delete "+opts.Entity)
	add(opts.Entity+"DeletionConfirmationMessage", "Are you sure you want to delete this "+opts.Entity+"?")
	add("Permission:"+permissionGroupName(info), permissionGroupName(info))
	add("Permission:"+opts.Entity, opts.Entity)
	add("Permission:"+opts.Entity+".Create", "Create "+opts.Entity)
	add("Permission:"+opts.Entity+".Update", "Update "+opts.Entity)
	add("Permission:"+opts.Entity+".Delete", "Delete "+opts.Entity)
	for _, attr := range opts.Attrs {
		add(opts.Entity+"."+attr.Name, splitPascal(attr.Name))
	}

	doc["texts"] = texts
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", false, err
	}
	return string(out) + "\n", changed, nil
}

func insertBeforeMethodClosingBrace(content string, methodName string, insert string) (string, bool) {
	methodIdx := strings.Index(content, methodName+"(")
	if methodIdx < 0 {
		methodIdx = strings.Index(content, methodName+" (")
	}
	if methodIdx < 0 {
		return content, false
	}
	openIdx := strings.Index(content[methodIdx:], "{")
	if openIdx < 0 {
		return content, false
	}
	openIdx += methodIdx
	closeIdx := matchingBrace(content, openIdx)
	if closeIdx < 0 {
		return content, false
	}
	return content[:closeIdx] + insert + content[closeIdx:], true
}

func insertAfterClassOpeningBrace(content string, className string, insert string) (string, bool) {
	if className == "" {
		return content, false
	}
	re := regexp.MustCompile(`\b(?:class|interface)\s+` + regexp.QuoteMeta(className) + `\b`)
	match := re.FindStringIndex(content)
	if match == nil {
		return content, false
	}
	openRelative := strings.Index(content[match[1]:], "{")
	if openRelative < 0 {
		return content, false
	}
	insertAt := match[1] + openRelative + 1
	return content[:insertAt] + "\n" + insert + content[insertAt:], true
}

func insertOnModelCreatingLine(content string, insert string) (string, bool) {
	methodIdx := strings.Index(content, "OnModelCreating(")
	if methodIdx < 0 {
		return content, false
	}
	openIdx := strings.Index(content[methodIdx:], "{")
	if openIdx < 0 {
		return content, false
	}
	openIdx += methodIdx
	baseCall := "base.OnModelCreating(builder);"
	if baseIdx := strings.Index(content[openIdx:], baseCall); baseIdx >= 0 {
		insertAt := openIdx + baseIdx + len(baseCall)
		lineEnd := strings.Index(content[insertAt:], "\n")
		if lineEnd >= 0 {
			insertAt += lineEnd + 1
		}
		return content[:insertAt] + insert + content[insertAt:], true
	}
	lineEnd := strings.Index(content[openIdx:], "\n")
	if lineEnd < 0 {
		return content[:openIdx+1] + "\n" + insert + content[openIdx+1:], true
	}
	insertAt := openIdx + lineEnd + 1
	return content[:insertAt] + insert + content[insertAt:], true
}

func matchingBrace(content string, openIdx int) int {
	depth := 0
	for i := openIdx; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func ensureUsing(content string, namespace string) string {
	usingLine := "using " + namespace + ";"
	if strings.Contains(content, usingLine) {
		return content
	}
	lines := strings.SplitAfter(content, "\n")
	insertAt := 0
	seenUsing := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "using ") {
			insertAt += len(line)
			seenUsing = true
			continue
		}
		if trimmed == "" && !seenUsing {
			insertAt += len(line)
			continue
		}
		break
	}
	if insertAt == 0 {
		return usingLine + "\n" + content
	}
	return content[:insertAt] + usingLine + "\n" + content[insertAt:]
}

func typeMemberIndent(content string) string {
	fileScopedNamespace := regexp.MustCompile(`(?m)^\s*namespace\s+[A-Za-z_][A-Za-z0-9_.]*\s*;`)
	if fileScopedNamespace.MatchString(content) {
		return "    "
	}
	return "        "
}

func inferKeyType(entityType string) string {
	start := strings.Index(entityType, "<")
	end := strings.LastIndex(entityType, ">")
	if start < 0 || end <= start+1 {
		return ""
	}
	return strings.TrimSpace(entityType[start+1 : end])
}

func parseMajor(version string) int {
	if version == "" {
		return 0
	}
	var digits strings.Builder
	for _, r := range version {
		if !unicode.IsDigit(r) {
			break
		}
		digits.WriteRune(r)
	}
	major, _ := strconv.Atoi(digits.String())
	return major
}

func compareVersion(a, b string) int {
	ap := versionParts(a)
	bp := versionParts(b)
	for i := 0; i < len(ap) || i < len(bp); i++ {
		var av, bv int
		if i < len(ap) {
			av = ap[i]
		}
		if i < len(bp) {
			bv = bp[i]
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	return 0
}

func versionParts(version string) []int {
	raw := strings.FieldsFunc(version, func(r rune) bool {
		return r == '.' || r == '-' || r == '+'
	})
	var out []int
	for _, part := range raw {
		value, err := strconv.Atoi(part)
		if err != nil {
			break
		}
		out = append(out, value)
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func toPascal(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	if len(parts) == 0 {
		return value
	}
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}

func lowerFirst(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func isCSharpIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func pluralize(word string) string {
	lower := strings.ToLower(word)
	switch {
	case strings.HasSuffix(lower, "ch"), strings.HasSuffix(lower, "sh"), strings.HasSuffix(lower, "ss"), strings.HasSuffix(lower, "x"), strings.HasSuffix(lower, "z"):
		return word + "es"
	case strings.HasSuffix(lower, "y") && len(word) > 1 && !isVowel(rune(lower[len(lower)-2])):
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(lower, "fe"):
		return word[:len(word)-2] + "ves"
	case strings.HasSuffix(lower, "f"):
		return word[:len(word)-1] + "ves"
	default:
		return word + "s"
	}
}

func isVowel(r rune) bool {
	return r == 'a' || r == 'e' || r == 'i' || r == 'o' || r == 'u'
}

func toKebab(value string) string {
	var out strings.Builder
	var previousLower bool
	for i, r := range value {
		if unicode.IsUpper(r) {
			if i > 0 && previousLower {
				out.WriteRune('-')
			}
			out.WriteRune(unicode.ToLower(r))
			previousLower = false
			continue
		}
		if r == '_' || r == ' ' {
			out.WriteRune('-')
			previousLower = false
			continue
		}
		out.WriteRune(unicode.ToLower(r))
		previousLower = unicode.IsLower(r) || unicode.IsDigit(r)
	}
	return out.String()
}

func splitPascal(value string) string {
	var out strings.Builder
	for i, r := range value {
		if i > 0 && unicode.IsUpper(r) {
			out.WriteRune(' ')
		}
		out.WriteRune(r)
	}
	return out.String()
}

func lastNamespaceSegment(namespace string) string {
	if idx := strings.LastIndex(namespace, "."); idx >= 0 {
		return namespace[idx+1:]
	}
	return namespace
}

func codePrefix(info AppInfo) string {
	if info.DbContextPrefix != "" {
		return info.DbContextPrefix
	}
	return lastNamespaceSegment(info.Module)
}

func permissionClassName(info AppInfo) string {
	return codePrefix(info) + "Permissions"
}

func permissionDefinitionProviderClassName(info AppInfo) string {
	return codePrefix(info) + "PermissionDefinitionProvider"
}

func permissionGroupName(info AppInfo) string {
	return codePrefix(info)
}
