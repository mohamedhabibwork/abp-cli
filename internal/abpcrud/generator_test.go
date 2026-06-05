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

func TestParseAttributeSupportsFilteringAndRelationModifiers(t *testing.T) {
	t.Parallel()

	attr, err := ParseAttribute("Guid CategoryId required filterable relation:Category")
	if err != nil {
		t.Fatal(err)
	}

	if attr.Name != "CategoryId" || attr.Type != "Guid" || !attr.Required {
		t.Fatalf("attribute parsed incorrectly: %+v", attr)
	}
	if attr.Filterable == nil || !*attr.Filterable {
		t.Fatalf("Filterable = %#v, want true", attr.Filterable)
	}
	if attr.Relation.Kind != relationReference || attr.Relation.Entity != "Category" {
		t.Fatalf("Relation = %+v, want reference Category", attr.Relation)
	}
}

func TestLoadOptionsFromJSONSupportsRelationMetadata(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "relation-crud.json")
	content := []byte(`{
  "entity": "Product",
  "attributes": [
    {
      "type": "Guid",
      "name": "categoryId",
      "required": true,
      "filterable": true,
      "relation": {
        "kind": "reference",
        "entity": "Category",
        "inverse": "Products"
      }
    }
  ]
}`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	opts, err := LoadOptionsFromJSON(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.Attrs) != 1 {
		t.Fatalf("Attrs length = %d, want 1", len(opts.Attrs))
	}
	attr := opts.Attrs[0]
	if attr.Filterable == nil || !*attr.Filterable {
		t.Fatalf("Filterable = %#v, want true", attr.Filterable)
	}
	if attr.Relation.Kind != relationReference || attr.Relation.Entity != "Category" || attr.Relation.Inverse != "Products" {
		t.Fatalf("Relation = %+v, want reference Category with Products inverse", attr.Relation)
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
		fileTypeGetListInput,
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
		fileTypeGetListInput,
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

func TestBuildPlanGeneratesFilteredListInputAndQuery(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	plan, err := BuildPlan(Options{
		Root:   root,
		Entity: "Product",
		Files:  []string{"contracts,application,httpapi"},
		Attrs: []Attribute{
			{Type: "string", Name: "Name", Required: true, MaxLength: 100},
			{Type: "decimal", Name: "Price"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	input := findChange(plan, fileTypeGetListInput)
	if input == nil {
		t.Fatal("missing get-list-input change")
	}
	inputContent := string(input.Content)
	for _, want := range []string{
		"class GetProductListInput : PagedAndSortedResultRequestDto",
		"public string? Filter { get; set; }",
		"public string? Name { get; set; }",
		"public decimal? Price { get; set; }",
	} {
		if !contains(inputContent, want) {
			t.Fatalf("list input did not contain %q:\n%s", want, inputContent)
		}
	}

	appService := findChange(plan, fileTypeAppService)
	if appService == nil {
		t.Fatal("missing appservice change")
	}
	appContent := string(appService.Content)
	for _, want := range []string{
		"GetListAsync(GetProductListInput input)",
		".WhereIf(!input.Filter.IsNullOrWhiteSpace()",
		".WhereIf(!input.Name.IsNullOrWhiteSpace(), x => x.Name.Contains(input.Name!))",
		".WhereIf(input.Price.HasValue, x => x.Price == input.Price!.Value)",
		".OrderBy(input.Sorting ?? ProductConstants.DefaultSorting)",
	} {
		if !contains(appContent, want) {
			t.Fatalf("app service did not contain %q:\n%s", want, appContent)
		}
	}
}

func TestBuildPlanUsesDomainValidationChecks(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	plan, err := BuildPlan(Options{
		Root:   root,
		Entity: "Product",
		Files:  []string{"entity"},
		Attrs: []Attribute{
			{Type: "string", Name: "Name", Required: true, MaxLength: 100},
			{Type: "decimal", Name: "Price", Required: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	entity := findChange(plan, fileTypeEntity)
	if entity == nil {
		t.Fatal("missing entity change")
	}
	content := string(entity.Content)
	if !contains(content, "Name = Check.NotNullOrWhiteSpace(name, nameof(name), maxLength: 100);") {
		t.Fatalf("entity did not use string Check validation:\n%s", content)
	}
	if !contains(content, "Price = price;") {
		t.Fatalf("entity should keep non-string assignment simple:\n%s", content)
	}
}

func TestBuildPlanGeneratesReferenceRelationCode(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	plan, err := BuildPlan(Options{
		Root:   root,
		Entity: "Product",
		Files:  []string{"all"},
		Attrs: []Attribute{
			{Type: "Guid", Name: "CategoryId", Required: true, Relation: Relation{Kind: relationReference, Entity: "Category", Inverse: "Products"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	entity := findChange(plan, fileTypeEntity)
	if entity == nil {
		t.Fatal("missing entity change")
	}
	entityContent := string(entity.Content)
	if !contains(entityContent, "public Guid CategoryId { get; private set; }") ||
		!contains(entityContent, "public Category? Category { get; private set; }") {
		t.Fatalf("entity did not include reference FK/navigation:\n%s", entityContent)
	}

	config := findChange(plan, fileTypeEFConfiguration)
	if config == nil {
		t.Fatal("missing ef configuration change")
	}
	configContent := string(config.Content)
	if !contains(configContent, "builder.HasOne(x => x.Category).WithMany(x => x.Products).HasForeignKey(x => x.CategoryId);") {
		t.Fatalf("EF configuration did not include reference relation:\n%s", configContent)
	}
}

func TestRelationMetadataRequiresExplicitInverseAndJoinDetails(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	_, err := BuildPlan(Options{
		Root:   root,
		Entity: "Product",
		Attrs: []Attribute{
			{Type: "ICollection<Category>", Name: "Categories", Relation: Relation{Kind: relationCollection, Entity: "Category"}},
		},
	})
	if err == nil || !contains(err.Error(), "requires inverse navigation metadata") {
		t.Fatalf("collection relation error = %v, want inverse metadata error", err)
	}

	_, err = BuildPlan(Options{
		Root:   root,
		Entity: "Product",
		Attrs: []Attribute{
			{Type: "ICollection<Tag>", Name: "Tags", Relation: Relation{Kind: relationManyToMany, Entity: "Tag", Inverse: "Products"}},
		},
	})
	if err == nil || !contains(err.Error(), "requires inverse, joinEntity, thisKey, and otherKey metadata") {
		t.Fatalf("many-to-many relation error = %v, want explicit metadata error", err)
	}
}

func TestBuildPlanGeneratesManyToManyJoinFiles(t *testing.T) {
	t.Parallel()

	root := createMinimalABPSolution(t)
	plan, err := BuildPlan(Options{
		Root:   root,
		Entity: "Product",
		Files:  []string{"all"},
		Attrs: []Attribute{
			{
				Type: "ICollection<Tag>",
				Name: "Tags",
				Relation: Relation{
					Kind:       relationManyToMany,
					Entity:     "Tag",
					Inverse:    "Products",
					JoinEntity: "ProductTag",
					ThisKey:    "ProductId",
					OtherKey:   "TagId",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	joinEntity := findChange(plan, fileTypeManyToManyJoinEntity)
	if joinEntity == nil {
		t.Fatal("missing many-to-many join entity change")
	}
	if !contains(string(joinEntity.Content), "class ProductTag : Entity") {
		t.Fatalf("join entity content unexpected:\n%s", string(joinEntity.Content))
	}

	joinConfig := findChange(plan, fileTypeManyToManyJoinConfiguration)
	if joinConfig == nil {
		t.Fatal("missing many-to-many join configuration change")
	}
	if !contains(string(joinConfig.Content), "builder.HasKey(x => new { x.ProductId, x.TagId });") {
		t.Fatalf("join configuration content unexpected:\n%s", string(joinConfig.Content))
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
