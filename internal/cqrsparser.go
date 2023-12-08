package internal

import (
	"log"
	"path"
	"regexp"
	"strings"
	"unicode"
)

type annotation string

const (
	GORS           annotation = "@GORS"
	CQRS           annotation = "@CQRS"
	Query          annotation = "@Query"
	Command        annotation = "@Command"
	QueryPath      annotation = "@QueryPath"
	CommandPath    annotation = "@CommandPath"
	QueryBusPath   annotation = "@QueryBusPath"
	CommandBusPath annotation = "@CommandBusPath"
	NamePrefix     annotation = "@NamePrefix"
	AssemblerPath  annotation = "@AssemblerPath"
	ServicePath    annotation = "@ServicePath"
	GOBasePath     annotation = "@GoBasePath"
)

func (a annotation) String() string {
	return string(a)
}

func (a annotation) EqualsIgnoreCase(str string) bool {
	return strings.ToUpper(str) == strings.ToUpper(a.String())
}

func (a annotation) PrefixOf(str string) bool {
	return strings.HasPrefix(strings.ToUpper(str), strings.ToUpper(a.String()))
}

type Path struct {
	ServiceImplPath string
	GoBasePath      string
	Query           string
	Command         string
	NamePrefix      string
	BusQuery        string
	BusCommand      string
	AssemblerPath   string
}

func NewPath(comments []string) *Path {
	info := &Path{}
	for _, comment := range comments {
		text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(comment), "//"))
		seg := strings.Split(text, " ")
		// 注释的开始必须以 @CQRS 开头
		if CQRS.EqualsIgnoreCase(seg[0]) {
			for _, s := range seg {
				s = strings.TrimSpace(s)
				switch {
				case QueryPath.PrefixOf(s):
					v, ok := ExtractValue(s, string(QueryPath))
					if !ok {
						log.Fatalf("error: %s query path invalid", s)
					}
					info.Query = v
				case CommandPath.PrefixOf(s):
					v, ok := ExtractValue(s, string(CommandPath))
					if !ok {
						log.Fatalf("error: %s command path invalid", s)
					}
					info.Command = v
				case NamePrefix.PrefixOf(s):
					v, ok := ExtractValue(s, string(NamePrefix))
					if !ok {
						log.Fatalf("error: %s NamePrefix invalid", s)
					}
					info.NamePrefix = v
				case QueryBusPath.PrefixOf(s):
					v, ok := ExtractValue(s, string(QueryBusPath))
					if !ok {
						log.Fatalf("error: %s QueryBusPath invalid", s)
					}
					info.BusQuery = v
				case CommandBusPath.PrefixOf(s):
					v, ok := ExtractValue(s, string(CommandBusPath))
					if !ok {
						log.Fatalf("error: %s CommandBusPath invalid", s)
					}
					info.BusCommand = v
				case AssemblerPath.PrefixOf(s):
					v, ok := ExtractValue(s, string(AssemblerPath))
					if !ok {
						log.Fatalf("error: %s AssemblerPath invalid", s)
					}
					info.AssemblerPath = v
				}
			}
		} else if GORS.EqualsIgnoreCase(seg[0]) {
			for _, s := range seg {
				s = strings.TrimSpace(s)
				switch {
				case ServicePath.PrefixOf(s):
					v, ok := ExtractValue(s, string(ServicePath))
					if !ok {
						log.Fatalf("error: %s query path invalid", s)
					}
					info.ServiceImplPath = v
				case GOBasePath.PrefixOf(s):
					v, ok := ExtractValue(s, string(GOBasePath))
					if !ok {
						log.Fatalf("error: %s query path invalid", s)
					}
					info.GoBasePath = v
				}
			}
		}
	}
	return info
}

func NewFileFromComment(
	endpoint string, queryDir, commandDir, queryRela, commandRela string, comments []string, NamePrefix string) *CQRSFile {

	for _, comment := range comments {
		text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(comment), "//"))
		seg := strings.Split(text, " ")
		// 注释的开始必须以 @CQRS 开头
		if !CQRS.EqualsIgnoreCase(seg[0]) {
			continue
		}
		for _, s := range seg {
			s = strings.TrimSpace(s)
			switch {
			case Query.EqualsIgnoreCase(s):
				return NewQueryFile(endpoint, queryDir, queryRela, NamePrefix)
			case Command.EqualsIgnoreCase(s):
				return NewCommandFile(endpoint, commandDir, commandRela, NamePrefix)
			}
		}
	}
	return nil
}

func NewQueryFile(endpoint string, queryDir, relaPath string, prefix string) *CQRSFile {
	fn := strings.ToLower(addUnderscore(endpoint)) + ".go"
	if prefix != "" {
		fn = prefix + "_" + fn
	}
	r := &CQRSFile{
		Type:          "query",
		RelaPath:      relaPath,
		AbsFilename:   path.Join(queryDir, fn),
		Package:       path.Base(queryDir),
		Endpoint:      endpoint,
		LowerEndpoint: strings.ToLower(endpoint[:1]) + endpoint[1:],
	}
	return r
}

func NewCommandFile(endpoint string, commandDir, commandRela string, prefix string) *CQRSFile {
	fn := strings.ToLower(addUnderscore(endpoint)) + ".go"
	if prefix != "" {
		fn = prefix + "_" + fn
	}
	r := &CQRSFile{
		Type:          "command",
		RelaPath:      commandRela,
		AbsFilename:   path.Join(commandDir, fn),
		Package:       path.Base(commandDir),
		Endpoint:      endpoint,
		LowerEndpoint: strings.ToLower(endpoint[:1]) + endpoint[1:],
	}
	return r
}

func addUnderscore(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(rune(s[i-1])) {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return result.String()
}

func ExtractValue(s string, annotation string) (string, bool) {
	reg := regexp.MustCompile(annotation + `\((.*)\)`)
	if !reg.MatchString(s) {
		return "", false
	}
	matchArr := reg.FindStringSubmatch(s)
	return matchArr[len(matchArr)-1], true
}
