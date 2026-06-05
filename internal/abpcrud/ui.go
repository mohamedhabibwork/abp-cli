package abpcrud

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

func RunInteractive(opts Options, stdin io.Reader, stdout io.Writer) (Options, error) {
	reader, ok := stdin.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(stdin)
	}

	root, err := resolveABPRoot(opts.Root)
	if err != nil {
		return Options{}, err
	}
	opts.Root = root

	modules, _ := detectModules(root)
	if len(modules) == 0 {
		return Options{}, fmt.Errorf("no ABP module layer projects were detected from %s or its parent directories", root)
	}

	fmt.Fprintln(stdout, "ABP CRUD Wizard")
	fmt.Fprintln(stdout, "===============")
	fmt.Fprintf(stdout, "Root: %s\n", root)
	if solution := findFirstFile(root, ".sln"); solution != "" {
		fmt.Fprintf(stdout, "Solution: %s\n", solution)
	}

	opts.Module, err = promptModule(reader, stdout, opts.Module, modules, root)
	if err != nil {
		return Options{}, err
	}
	fmt.Fprintf(stdout, "Module: %s\n", opts.Module)

	opts.Entity, err = promptRequired(reader, stdout, "Entity name", opts.Entity)
	if err != nil {
		return Options{}, err
	}
	opts.Entity = toPascal(opts.Entity)

	opts.EntityType, err = promptValue(reader, stdout, "Entity base type", firstNonEmpty(opts.EntityType, defaultEntityType))
	if err != nil {
		return Options{}, err
	}

	defaultKey := opts.KeyType
	if defaultKey == "" {
		defaultKey = inferKeyType(opts.EntityType)
	}
	opts.KeyType, err = promptValue(reader, stdout, "Key type", defaultKey)
	if err != nil {
		return Options{}, err
	}

	opts.Mapping, err = promptMapping(reader, stdout, opts.Mapping)
	if err != nil {
		return Options{}, err
	}

	opts.Files, err = promptFiles(reader, stdout, opts.Files)
	if err != nil {
		return Options{}, err
	}

	opts.Attrs, err = promptAttributes(reader, stdout, opts.Attrs)
	if err != nil {
		return Options{}, err
	}

	fmt.Fprintln(stdout)
	return opts, nil
}

func promptModule(reader *bufio.Reader, stdout io.Writer, requested string, modules map[string]map[string]string, root string) (string, error) {
	if requested != "" {
		if _, ok := modules[requested]; !ok {
			return "", fmt.Errorf("module %q was not found under %s", requested, filepath.Join(root, "src"))
		}
		return requested, nil
	}

	solutionPath := findFirstFile(root, ".sln")
	if module, err := chooseModule(modules, solutionPath); err == nil {
		return module, nil
	}

	names := sortedKeys(modules)
	if len(names) == 1 {
		return names[0], nil
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Detected modules:")
	for i, name := range names {
		fmt.Fprintf(stdout, "  %d. %s\n", i+1, name)
	}

	for {
		answer, err := promptValue(reader, stdout, "Module number", "1")
		if err != nil {
			return "", err
		}
		index, parseErr := strconv.Atoi(answer)
		if parseErr == nil && index >= 1 && index <= len(names) {
			return names[index-1], nil
		}
		fmt.Fprintf(stdout, "Choose a number from 1 to %d.\n", len(names))
	}
}

func promptMapping(reader *bufio.Reader, stdout io.Writer, current string) (string, error) {
	defaultValue := normalizeMappingStyle(current)
	if defaultValue == "" {
		defaultValue = mappingAuto
	}

	for {
		value, err := promptValue(reader, stdout, "Mapping style (auto, automapper, mapperly)", defaultValue)
		if err != nil {
			return "", err
		}
		if normalized := normalizeMappingStyle(value); normalized != "" {
			return normalized, nil
		}
		fmt.Fprintln(stdout, "Use auto, automapper, or mapperly.")
	}
}

func promptFiles(reader *bufio.Reader, stdout io.Writer, current []string) ([]string, error) {
	defaultValue := "all"
	if len(current) > 0 {
		defaultValue = strings.Join(current, ",")
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "File groups: all, domain-shared, domain, contracts, application, httpapi, efcore, localization")
	fmt.Fprintln(stdout, "Useful aliases: dto, repository, permissions, mapping, controller, dbcontexts")

	for {
		value, err := promptValue(reader, stdout, "Files", defaultValue)
		if err != nil {
			return nil, err
		}
		files, normalizeErr := normalizeFileTypes([]string{value})
		if normalizeErr == nil {
			return files, nil
		}
		fmt.Fprintf(stdout, "%v\n", normalizeErr)
	}
}

func promptAttributes(reader *bufio.Reader, stdout io.Writer, attrs []Attribute) ([]Attribute, error) {
	if len(attrs) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Attributes already loaded:")
		for _, attr := range attrs {
			fmt.Fprintf(stdout, "  - %s\n", formatAttribute(attr))
		}
		addMore, err := promptYesNo(reader, stdout, "Add more attributes", false)
		if err != nil {
			return nil, err
		}
		if !addMore {
			return attrs, nil
		}
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, `Attribute format: string Name required maxlen:100 filterable`)
	fmt.Fprintln(stdout, `Relation examples: Guid CategoryId required relation:Category, Guid TagIds relation:Tag:many-to-many`)
	fmt.Fprintln(stdout, "Leave the attribute prompt empty when finished.")

	for {
		value, err := promptLine(reader, stdout, "Attribute", "")
		if err != nil {
			return nil, err
		}
		if value == "" {
			return attrs, nil
		}
		attr, parseErr := ParseAttribute(value)
		if parseErr != nil {
			fmt.Fprintf(stdout, "Invalid attribute: %v\n", parseErr)
			continue
		}
		attrs = append(attrs, attr)
	}
}

func promptRequired(reader *bufio.Reader, stdout io.Writer, label string, defaultValue string) (string, error) {
	for {
		value, err := promptValue(reader, stdout, label, defaultValue)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(value) != "" {
			return value, nil
		}
		fmt.Fprintf(stdout, "%s is required.\n", label)
	}
}

func promptValue(reader *bufio.Reader, stdout io.Writer, label string, defaultValue string) (string, error) {
	return promptLine(reader, stdout, label, defaultValue)
}

func promptYesNo(reader *bufio.Reader, stdout io.Writer, label string, defaultValue bool) (bool, error) {
	defaultText := "n"
	if defaultValue {
		defaultText = "y"
	}

	for {
		value, err := promptLine(reader, stdout, label+" (y/n)", defaultText)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Fprintln(stdout, "Use y or n.")
		}
	}
}

func promptLine(reader *bufio.Reader, stdout io.Writer, label string, defaultValue string) (string, error) {
	if defaultValue == "" {
		fmt.Fprintf(stdout, "%s: ", label)
	} else {
		fmt.Fprintf(stdout, "%s [%s]: ", label, defaultValue)
	}

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if err != nil && errors.Is(err, io.EOF) && line == "" {
		return "", io.ErrUnexpectedEOF
	}

	value := strings.TrimSpace(line)
	if value == "" {
		value = defaultValue
	}
	return value, nil
}

func formatAttribute(attr Attribute) string {
	var parts []string
	parts = append(parts, attr.Type, attr.Name)
	if attr.Required {
		parts = append(parts, "required")
	}
	if attr.MaxLength > 0 {
		parts = append(parts, fmt.Sprintf("maxlen:%d", attr.MaxLength))
	}
	if attr.ForeignKey {
		parts = append(parts, "foreignkey")
	}
	if attr.Filterable != nil && *attr.Filterable {
		parts = append(parts, "filterable")
	}
	if attr.Relation.Kind != "" && attr.Relation.Entity != "" {
		parts = append(parts, "relation:"+attr.Relation.Entity+":"+attr.Relation.Kind)
	}
	return strings.Join(parts, " ")
}
