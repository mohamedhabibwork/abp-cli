package abpcrud

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"text/template"
)

//go:embed default_templates/*
var defaultTemplateFS embed.FS

const defaultTemplateDir = "default_templates"

type templateContext struct {
	Info    AppInfo
	Options Options

	Module       string
	Entity       string
	EntityType   string
	KeyType      string
	Attrs        []Attribute
	PluralEntity string
	Route        string

	PermissionsClass                  string
	PermissionDefinitionProviderClass string
	PermissionGroup                   string
	EntityPermissionVariable          string

	ResourceType      string
	ResourceNamespace string

	DbContextName         string
	DbContextPrefix       string
	DbPropertiesName      string
	DbPropertiesNamespace string
	DbPropertiesUsing     string

	AttributeUsingBlock    string
	ConstantsValidations   string
	EtoProperties          string
	EntityProperties       string
	InputProperties        string
	ReadDtoProperties      string
	GetListInputName       string
	GetListInputProperties string
	FilterPredicates       string
	ConstructorArgs        string
	ConstructorSets        string
	EntitySetters          string
	UpdateArgs             string

	DtoName              string
	DtoBaseClass         string
	AppServiceCreateArgs string
	AppServiceUpdateArgs string

	EFPropertyConfig     string
	EFRelationshipConfig string
	ManyToManyJoinEntity string
	ManyToManyThisKey    string
	ManyToManyOtherKey   string
	Culture              string
}

type templateOptions struct {
	DtoName string
	Culture string
}

func renderABPTemplate(name string, info AppInfo, opts Options, templateOpts templateOptions) (string, error) {
	source, sourceLabel, err := loadTemplateSource(opts.TemplateDir, name)
	if err != nil {
		return "", err
	}

	parsed, err := template.New(name).
		Option("missingkey=error").
		Funcs(templateFuncs()).
		Parse(source)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", sourceLabel, err)
	}

	var out bytes.Buffer
	if err := parsed.Execute(&out, buildTemplateContext(info, opts, templateOpts)); err != nil {
		return "", fmt.Errorf("execute template %s: %w", sourceLabel, err)
	}
	return out.String(), nil
}

func loadTemplateSource(templateDir string, name string) (string, string, error) {
	if templateDir != "" {
		path := filepath.Join(templateDir, name)
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content), path, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", "", fmt.Errorf("read custom template %s: %w", path, err)
		}
	}

	path := filepath.Join(defaultTemplateDir, name)
	content, err := defaultTemplateFS.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read built-in template %s: %w", name, err)
	}
	return string(content), "built-in:" + name, nil
}

func buildTemplateContext(info AppInfo, opts Options, templateOpts templateOptions) templateContext {
	dbPropertiesUsing := ""
	if info.DbPropertiesNamespace != "" {
		dbPropertiesUsing = "using " + info.DbPropertiesNamespace + ";\n"
	}

	return templateContext{
		Info:    info,
		Options: opts,

		Module:       info.Module,
		Entity:       opts.Entity,
		EntityType:   opts.EntityType,
		KeyType:      opts.KeyType,
		Attrs:        opts.Attrs,
		PluralEntity: pluralize(opts.Entity),
		Route:        "api/app/" + toKebab(pluralize(opts.Entity)),

		PermissionsClass:                  permissionClassName(info),
		PermissionDefinitionProviderClass: permissionDefinitionProviderClassName(info),
		PermissionGroup:                   permissionGroupName(info),
		EntityPermissionVariable:          lowerFirst(opts.Entity) + "Permission",

		ResourceType:      info.ResourceType,
		ResourceNamespace: info.ResourceNamespace,

		DbContextName:         info.DbContextName,
		DbContextPrefix:       info.DbContextPrefix,
		DbPropertiesName:      info.DbPropertiesName,
		DbPropertiesNamespace: info.DbPropertiesNamespace,
		DbPropertiesUsing:     dbPropertiesUsing,

		AttributeUsingBlock:    attributeUsingBlock(opts.Attrs),
		ConstantsValidations:   constantsValidationBlock(opts),
		EtoProperties:          etoPropertiesBlock(opts),
		EntityProperties:       entityPropertiesBlock(opts),
		InputProperties:        inputPropertiesBlock(opts),
		ReadDtoProperties:      readDtoPropertiesBlock(opts),
		GetListInputName:       getListInputName(opts),
		GetListInputProperties: getListInputPropertiesBlock(opts),
		FilterPredicates:       filterPredicatesBlock(opts),
		ConstructorArgs:        constructorArgs(opts),
		ConstructorSets:        constructorSetsBlock(opts),
		EntitySetters:          entitySettersBlock(opts),
		UpdateArgs:             updateArgs(opts),

		DtoName:              templateOpts.DtoName,
		DtoBaseClass:         dtoBaseClass(opts.EntityType),
		AppServiceCreateArgs: appServiceCreateArgs(opts),
		AppServiceUpdateArgs: appServiceUpdateArgs(opts),

		EFPropertyConfig:     efPropertyConfigBlock(opts),
		EFRelationshipConfig: efRelationshipConfigBlock(opts),
		ManyToManyJoinEntity: firstManyToManyRelation(opts).JoinEntity,
		ManyToManyThisKey:    firstManyToManyRelation(opts).ThisKey,
		ManyToManyOtherKey:   firstManyToManyRelation(opts).OtherKey,
		Culture:              templateOpts.Culture,
	}
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"lowerFirst":  lowerFirst,
		"pluralize":   pluralize,
		"toKebab":     toKebab,
		"splitPascal": splitPascal,
	}
}

func listDefaultTemplateNames() ([]string, error) {
	entries, err := fs.ReadDir(defaultTemplateFS, defaultTemplateDir)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func writeDefaultTemplates(outputDir string, overwrite bool) ([]FileChange, error) {
	names, err := listDefaultTemplateNames()
	if err != nil {
		return nil, err
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absOutputDir, 0o755); err != nil {
		return nil, err
	}

	changes := make([]FileChange, 0, len(names))
	for _, name := range names {
		content, err := defaultTemplateFS.ReadFile(filepath.Join(defaultTemplateDir, name))
		if err != nil {
			return nil, err
		}

		path := filepath.Join(absOutputDir, name)
		action := "create"
		if existing, readErr := os.ReadFile(path); readErr == nil {
			if string(existing) == string(content) {
				action = "unchanged"
			} else if overwrite {
				action = "overwrite"
			} else {
				action = "skip-existing"
			}
		}

		if action == "create" || action == "overwrite" {
			if err := os.WriteFile(path, content, 0o644); err != nil {
				return nil, err
			}
		}

		changes = append(changes, FileChange{
			Type:    "template",
			Path:    path,
			Action:  action,
			Content: content,
		})
	}
	return changes, nil
}
