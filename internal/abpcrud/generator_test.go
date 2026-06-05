package abpcrud

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOptionsFromJSONSupportsStructuredAndStringAttributes(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(configDir, "product-crud.json")
	content := []byte(`{
  "root": "../app",
  "entity": "Product",
  "entity_type": "FullAuditedAggregateRoot<Guid>",
  "keyType": "Guid",
  "templateDir": "../templates",
  "files": ["entity", "dto"],
  "yes": true,
  "dryRun": true,
  "attributes": [
    {
      "type": "string",
      "name": "Name",
      "required": true,
      "maxLength": 100
    },
    "decimal Price required"
  ]
}`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	opts, err := LoadOptionsFromJSON(configPath)
	if err != nil {
		t.Fatal(err)
	}

	wantRoot := filepath.Join(configDir, "../app")
	if opts.Root != wantRoot {
		t.Fatalf("Root = %q, want %q", opts.Root, wantRoot)
	}
	if opts.Entity != "Product" {
		t.Fatalf("Entity = %q, want Product", opts.Entity)
	}
	if opts.EntityType != "FullAuditedAggregateRoot<Guid>" {
		t.Fatalf("EntityType = %q", opts.EntityType)
	}
	if opts.KeyType != "Guid" {
		t.Fatalf("KeyType = %q, want Guid", opts.KeyType)
	}
	wantTemplateDir := filepath.Join(configDir, "../templates")
	if opts.TemplateDir != wantTemplateDir {
		t.Fatalf("TemplateDir = %q, want %q", opts.TemplateDir, wantTemplateDir)
	}
	if !opts.AssumeYes {
		t.Fatal("AssumeYes = false, want true")
	}
	if !opts.DryRun {
		t.Fatal("DryRun = false, want true")
	}
	if len(opts.Files) != 2 || opts.Files[0] != "entity" || opts.Files[1] != "dto" {
		t.Fatalf("Files = %#v, want entity and dto", opts.Files)
	}
	if len(opts.Attrs) != 2 {
		t.Fatalf("Attrs length = %d, want 2", len(opts.Attrs))
	}
	if opts.Attrs[0].Name != "Name" || opts.Attrs[0].Type != "string" || !opts.Attrs[0].Required || opts.Attrs[0].MaxLength != 100 {
		t.Fatalf("first attr parsed incorrectly: %+v", opts.Attrs[0])
	}
	if opts.Attrs[1].Name != "Price" || opts.Attrs[1].Type != "decimal" || !opts.Attrs[1].Required {
		t.Fatalf("second attr parsed incorrectly: %+v", opts.Attrs[1])
	}
}

func TestRunVersionPrintsBuildMetadata(t *testing.T) {
	oldVersion := Version
	oldCommit := Commit
	oldBuildDate := BuildDate
	t.Cleanup(func() {
		Version = oldVersion
		Commit = oldCommit
		BuildDate = oldBuildDate
	})

	Version = "v9.8.7"
	Commit = "abc123"
	BuildDate = "2026-06-05T00:00:00Z"

	var stdout bytes.Buffer
	code := Run([]string{"version"}, strings.NewReader(""), &stdout, io.Discard)
	if code != 0 {
		t.Fatalf("Run(version) exit code = %d, want 0", code)
	}

	output := stdout.String()
	for _, want := range []string{"abp-cli v9.8.7", "commit: abc123", "built: 2026-06-05T00:00:00Z"} {
		if !strings.Contains(output, want) {
			t.Fatalf("version output %q does not contain %q", output, want)
		}
	}
}

func TestDetectMappingStylePrefersMapperlyMarkers(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "App.csproj")
	content := []byte(`<Project Sdk="Microsoft.NET.Sdk">
  <ItemGroup>
    <PackageReference Include="Volo.Abp.Core" Version="10.1.0" />
    <PackageReference Include="Volo.Abp.Mapperly" Version="10.1.0" />
  </ItemGroup>
</Project>`)
	if err := os.WriteFile(projectPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	style, source := detectMappingStyle(tempDir, 10)
	if style != mappingMapperly {
		t.Fatalf("style = %q, want %q (%s)", style, mappingMapperly, source)
	}
}

func TestDetectMappingStyleUsesVersionDefaults(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	style, _ := detectMappingStyle(tempDir, 10)
	if style != mappingMapperly {
		t.Fatalf("ABP 10 default style = %q, want %q", style, mappingMapperly)
	}

	style, _ = detectMappingStyle(tempDir, 9)
	if style != mappingAutoMapper {
		t.Fatalf("ABP 9 default style = %q, want %q", style, mappingAutoMapper)
	}
}

func TestNormalizeFileTypesSupportsGroupsAndAliases(t *testing.T) {
	t.Parallel()

	got, err := normalizeFileTypes([]string{"entity,dto", "Controller"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		fileTypeEntity,
		fileTypeCreateDto,
		fileTypeUpdateDto,
		fileTypeEntityDto,
		fileTypeController,
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%#v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q (%#v)", i, got[i], want[i], got)
		}
	}
}

func TestNormalizeFileTypesRejectsUnknownValues(t *testing.T) {
	t.Parallel()

	if _, err := normalizeFileTypes([]string{"entity,banana"}); err == nil {
		t.Fatal("normalizeFileTypes returned nil error for an unknown file type")
	}
}

func TestBuildPlanCanGenerateOnlySelectedFiles(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	plan, err := BuildPlan(Options{
		Root:   root,
		Entity: "Product",
		Files:  []string{"entity,dto,controller"},
		Attrs: []Attribute{
			{Type: "string", Name: "Name", Required: true, MaxLength: 100},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wantTypes := []string{
		fileTypeEntity,
		fileTypeCreateDto,
		fileTypeUpdateDto,
		fileTypeEntityDto,
		fileTypeController,
	}
	gotTypes := make([]string, 0, len(plan.Changes))
	for _, change := range plan.Changes {
		gotTypes = append(gotTypes, change.Type)
	}
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("change types = %#v, want %#v", gotTypes, wantTypes)
	}
	for _, want := range wantTypes {
		if !hasChangeType(plan, want) {
			t.Fatalf("missing change type %q in %#v", want, gotTypes)
		}
	}
	if hasChangeType(plan, fileTypeDbContext) || hasChangeType(plan, fileTypePermissions) {
		t.Fatalf("plan included unselected types: %#v", gotTypes)
	}
}

func TestDbContextOnlySelectionDoesNotReferenceMissingConfiguration(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	plan, err := BuildPlan(Options{
		Root:   root,
		Entity: "Product",
		Files:  []string{"dbcontext"},
		Attrs: []Attribute{
			{Type: "string", Name: "Name", Required: true, MaxLength: 100},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(plan.Changes) != 1 {
		t.Fatalf("changes = %d, want 1 (%#v)", len(plan.Changes), plan.Changes)
	}
	change := plan.Changes[0]
	if change.Type != fileTypeDbContext {
		t.Fatalf("change type = %q, want %q", change.Type, fileTypeDbContext)
	}
	content := string(change.Content)
	if !contains(content, "DbSet<Product> Products") {
		t.Fatalf("DbContext content did not include the DbSet:\n%s", content)
	}
	if contains(content, "ProductConfiguration") || contains(content, ".Configurations") {
		t.Fatalf("DbContext-only generation referenced a missing configuration file:\n%s", content)
	}
}

func TestEFCoreSelectionAddsConfigurationAndDbContextReference(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	plan, err := BuildPlan(Options{
		Root:   root,
		Entity: "Product",
		Files:  []string{"efcore"},
		Attrs: []Attribute{
			{Type: "string", Name: "Name", Required: true, MaxLength: 100},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !hasChangeType(plan, fileTypeEFConfiguration) {
		t.Fatal("missing EF configuration change")
	}
	dbContext := findChange(plan, fileTypeDbContext)
	if dbContext == nil {
		t.Fatal("missing DbContext change")
	}
	content := string(dbContext.Content)
	if !contains(content, "builder.ApplyConfiguration(new ProductConfiguration());") {
		t.Fatalf("DbContext content did not apply the generated configuration:\n%s", content)
	}
}

func TestGeneratedAppServiceUsesExplicitEntityConstruction(t *testing.T) {
	t.Parallel()

	content, err := appServiceTemplate(AppInfo{
		Module: "Acme.BookStore",
	}, Options{
		Entity:     "Product",
		EntityType: defaultEntityType,
		KeyType:    "Guid",
		Attrs: []Attribute{
			{Type: "string", Name: "Name", Required: true, MaxLength: 100},
			{Type: "decimal", Name: "Price", Required: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !contains(content, "public override async Task<ProductDto> CreateAsync(CreateProductDto input)") {
		t.Fatalf("CreateAsync override missing:\n%s", content)
	}
	if !contains(content, "var entity = new Product(GuidGenerator.Create(), input.Name, input.Price);") {
		t.Fatalf("explicit Product construction missing:\n%s", content)
	}
	if !contains(content, "entity.Update(input.Name, input.Price);") {
		t.Fatalf("explicit Product update missing:\n%s", content)
	}
}

func TestCustomTemplateDirectoryOverridesBuiltInTemplate(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	templateDir := t.TempDir()
	writeTestFile(t, filepath.Join(templateDir, templateEntity), `// custom entity for {{.Entity}}
namespace {{.Module}}.Domain.Entities
{
    public class {{.Entity}}
    {
    }
}
`)

	plan, err := BuildPlan(Options{
		Root:        root,
		Entity:      "Product",
		Files:       []string{"entity"},
		TemplateDir: templateDir,
		Attrs: []Attribute{
			{Type: "string", Name: "Name", Required: true, MaxLength: 100},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	change := findChange(plan, fileTypeEntity)
	if change == nil {
		t.Fatal("missing entity change")
	}
	content := string(change.Content)
	if !contains(content, "// custom entity for Product") {
		t.Fatalf("custom template was not used:\n%s", content)
	}
}

func TestCustomTemplateErrorsFailPlan(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	templateDir := t.TempDir()
	writeTestFile(t, filepath.Join(templateDir, templateEntity), `{{ .DoesNotExist }}`)

	_, err := BuildPlan(Options{
		Root:        root,
		Entity:      "Product",
		Files:       []string{"entity"},
		TemplateDir: templateDir,
	})
	if err == nil {
		t.Fatal("BuildPlan returned nil error for an invalid custom template")
	}
	if !contains(err.Error(), "render entity template") {
		t.Fatalf("error = %q, want render entity template context", err)
	}
}

func TestDetectAppWalksUpFromNestedProjectFolder(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	nested := filepath.Join(root, "src", "Acme.BookStore.Domain", "Entities")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	info, err := DetectApp(nested, "")
	if err != nil {
		t.Fatal(err)
	}

	if info.Root != root {
		t.Fatalf("Root = %q, want %q", info.Root, root)
	}
	if info.Module != "Acme.BookStore" {
		t.Fatalf("Module = %q, want Acme.BookStore", info.Module)
	}
}

func TestDetectAppInfersCurrentModuleInMultiModuleSolution(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	createLayerProjectDirs(t, root, "Acme.Inventory")
	nested := filepath.Join(root, "src", "Acme.Inventory.Domain", "Entities")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	info, err := DetectApp(nested, "")
	if err != nil {
		t.Fatal(err)
	}

	if info.Root != root {
		t.Fatalf("Root = %q, want %q", info.Root, root)
	}
	if info.Module != "Acme.Inventory" {
		t.Fatalf("Module = %q, want Acme.Inventory", info.Module)
	}
}

func TestRunInteractiveBuildsOptionsFromWizardInput(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	nested := filepath.Join(root, "src", "Acme.BookStore.EntityFrameworkCore", "EntityFrameworkCore")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader(strings.Join([]string{
		"Product",
		"",
		"",
		"",
		"entity,dto",
		"string Name required maxlen:100",
		"",
		"",
	}, "\n"))

	opts, err := RunInteractive(Options{Root: nested}, input, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if opts.Root != root {
		t.Fatalf("Root = %q, want %q", opts.Root, root)
	}
	if opts.Module != "Acme.BookStore" {
		t.Fatalf("Module = %q, want Acme.BookStore", opts.Module)
	}
	if opts.Entity != "Product" {
		t.Fatalf("Entity = %q, want Product", opts.Entity)
	}
	if opts.KeyType != "Guid" {
		t.Fatalf("KeyType = %q, want Guid", opts.KeyType)
	}
	if !hasString(opts.Files, fileTypeEntity) || !hasString(opts.Files, fileTypeCreateDto) || hasString(opts.Files, fileTypeController) {
		t.Fatalf("Files = %#v, want entity and dto only", opts.Files)
	}
	if len(opts.Attrs) != 1 || opts.Attrs[0].Name != "Name" || !opts.Attrs[0].Required || opts.Attrs[0].MaxLength != 100 {
		t.Fatalf("Attrs parsed incorrectly: %#v", opts.Attrs)
	}
}

func createLayerProjectDirs(t *testing.T, root string, module string) {
	t.Helper()

	layers := []string{
		layerDomainShared,
		layerDomain,
		layerApplicationContracts,
		layerApplication,
		layerHttpAPI,
		layerEntityFrameworkCore,
	}
	for _, layer := range layers {
		dir := filepath.Join(root, "src", module+"."+layer)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(dir, module+"."+layer+".csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <ItemGroup>
    <PackageReference Include="Volo.Abp.Core" Version="10.1.0" />
  </ItemGroup>
</Project>`)
	}
}

func createMinimalABPSolution(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	module := "Acme.BookStore"
	createLayerProjectDirs(t, root, module)
	writeTestFile(t, filepath.Join(root, "Acme.BookStore.sln"), "")

	domainShared := filepath.Join(root, "src", module+"."+layerDomainShared)
	writeTestFile(t, filepath.Join(domainShared, "Localization", "BookStoreResource.cs"), `namespace Acme.BookStore.Localization
{
    public class BookStoreResource
    {
    }
}
`)
	writeTestFile(t, filepath.Join(domainShared, "Localization", "BookStore", "en.json"), `{"culture":"en","texts":{}}`)
	writeTestFile(t, filepath.Join(domainShared, "Localization", "BookStore", "ar.json"), `{"culture":"ar","texts":{}}`)

	efCore := filepath.Join(root, "src", module+"."+layerEntityFrameworkCore, "EntityFrameworkCore")
	writeTestFile(t, filepath.Join(efCore, "BookStoreDbProperties.cs"), `namespace Acme.BookStore.EntityFrameworkCore
{
    public static class BookStoreDbProperties
    {
        public const string ConnectionStringName = "Default";
        public static string DbTablePrefix { get; set; } = "App";
        public static string? DbSchema { get; set; } = null;
    }
}
`)
	writeTestFile(t, filepath.Join(efCore, "BookStoreDbContext.cs"), `using Microsoft.EntityFrameworkCore;
using Volo.Abp.EntityFrameworkCore;

namespace Acme.BookStore.EntityFrameworkCore
{
    public class BookStoreDbContext : AbpDbContext<BookStoreDbContext>
    {
        public BookStoreDbContext(DbContextOptions<BookStoreDbContext> options)
            : base(options)
        {
        }

        protected override void OnModelCreating(ModelBuilder builder)
        {
            base.OnModelCreating(builder);
        }
    }
}
`)

	return root
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasChangeType(plan *Plan, fileType string) bool {
	return findChange(plan, fileType) != nil
}

func findChange(plan *Plan, fileType string) *FileChange {
	for i := range plan.Changes {
		if plan.Changes[i].Type == fileType {
			return &plan.Changes[i]
		}
	}
	return nil
}

func contains(value string, needle string) bool {
	return strings.Contains(value, needle)
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
