package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var version = "dev"

type resource struct {
	name, endpoint         string
	listFields, viewFields []string
}

var resources = map[string]resource{
	"issue":    {"issues", "issues", []string{"iid", "title", "state"}, []string{"iid", "title", "state", "author.username", "assignees", "labels", "description", "web_url"}},
	"mr":       {"merge_requests", "merge_requests", []string{"iid", "title", "state", "draft"}, []string{"iid", "title", "state", "author.username", "source_branch", "target_branch", "merge_status", "description", "web_url"}},
	"pipeline": {"pipelines", "pipelines", []string{"id", "status", "ref", "sha"}, []string{"id", "status", "ref", "sha", "source", "web_url", "created_at", "updated_at"}},
	"label":    {"labels", "labels", []string{"id", "name", "color", "description"}, []string{"id", "name", "color", "description"}},
}

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if has(args, "--version") || has(args, "-v") || has(args, "-V") {
		fmt.Println("version: " + version)
		return 0
	}
	if has(args, "--help") || has(args, "-h") {
		fmt.Print(help(args))
		return 0
	}
	repo, args, err := takeValue(args, "-R", "--repo")
	if err != nil {
		return usageError(err.Error(), "glab-axi <command> [--repo <group/project>]")
	}
	if len(args) == 0 {
		return dashboard(repo)
	}
	if args[0] == "help" {
		fmt.Print(help(args[1:]))
		return 0
	}
	if args[0] == "api" {
		return api(args[1:])
	}
	r, ok := resources[args[0]]
	if !ok {
		return usageError("unknown command: "+args[0], "glab-axi --help")
	}
	return resourceCommand(r, repo, args[1:])
}

func resourceCommand(r resource, repo string, args []string) int {
	action := "list"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		action, args = args[0], args[1:]
	}
	switch action {
	case "list":
		limit, fields, query, err := listArgs(args)
		if err != nil {
			return usageError(err.Error(), "glab-axi "+commandName(r)+" list [--state <state>] [--limit <n>] [--fields <a,b>]")
		}
		if fields == nil {
			fields = r.listFields
		}
		query = append(query, "per_page="+strconv.Itoa(limit))
		data, err := callAPI(projectEndpoint(repo, r.endpoint) + "?" + strings.Join(query, "&"))
		if err != nil {
			return commandError(err)
		}
		items, ok := data.([]any)
		if !ok {
			return commandError(errors.New("unexpected list response"))
		}
		if len(items) == 0 {
			fmt.Printf("%s: 0 found\n", r.name)
			return 0
		}
		fmt.Printf("count: %d returned\n%s", len(items), renderList(r.name, items, fields))
		fmt.Printf("\nhelp[1]: Run `glab-axi %s view <id>` for details\n", commandName(r))
		return 0
	case "view":
		if len(args) == 0 {
			return usageError("view requires an id", "glab-axi "+commandName(r)+" view <id>")
		}
		for _, arg := range args[1:] {
			if arg != "--full" {
				return usageError("unsupported flag: "+arg, "glab-axi "+commandName(r)+" view <id> [--full]")
			}
		}
		full := has(args[1:], "--full")
		data, err := callAPI(projectEndpoint(repo, r.endpoint) + "/" + url.PathEscape(args[0]))
		if err != nil {
			return commandError(err)
		}
		item, ok := data.(map[string]any)
		if !ok {
			return commandError(errors.New("unexpected detail response"))
		}
		item, truncated := selectFields(item, r.viewFields, full)
		fmt.Print(renderObject(singular(r.name), item))
		if truncated {
			fmt.Printf("\nhelp[1]: Run `glab-axi %s view %s --full` for complete text\n", commandName(r), args[0])
		}
		return 0
	default:
		return usageError("unknown subcommand: "+action, "glab-axi "+commandName(r)+" <list|view>")
	}
}

func dashboard(repo string) int {
	bin, _ := os.Executable()
	if home, _ := os.UserHomeDir(); home != "" {
		bin = strings.Replace(bin, home, "~", 1)
	}
	fmt.Printf("bin: %s\ndescription: Agent ergonomic interface for GitLab projects.\n", quote(bin))
	for _, key := range []string{"issue", "mr"} {
		r := resources[key]
		data, err := callAPI(projectEndpoint(repo, r.endpoint) + "?state=opened&per_page=10")
		if err != nil {
			return commandError(err)
		}
		items, _ := data.([]any)
		if len(items) == 0 {
			fmt.Printf("%s: 0 open\n", r.name)
			continue
		}
		fmt.Printf("%s", renderList(r.name, items, r.listFields))
	}
	fmt.Println("help[2]:\n  Run `glab-axi issue list` to list issues\n  Run `glab-axi mr list` to list merge requests")
	return 0
}

func api(args []string) int {
	if len(args) == 0 {
		return usageError("api requires an endpoint", "glab-axi api <endpoint> [glab api flags]")
	}
	data, err := callGlab(append([]string{"api"}, args...)...)
	if err != nil {
		return commandError(err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return commandError(errors.New("API returned non-JSON output"))
	}
	fmt.Println(encodeTOON(value))
	return 0
}

func callAPI(endpoint string) (any, error) {
	b, err := callGlab("api", endpoint)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(b, &value); err != nil {
		return nil, errors.New("API returned invalid JSON")
	}
	return value, nil
}

func callGlab(args ...string) ([]byte, error) {
	cmd := exec.Command("glab", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr, cmd.Stdin = &stdout, &stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		if i := strings.LastIndex(message, "ERROR:"); i >= 0 {
			message = strings.TrimSpace(message[i+6:])
		}
		return nil, errors.New(firstLine(message))
	}
	return stdout.Bytes(), nil
}

func projectEndpoint(repo, endpoint string) string {
	id := ":id"
	if repo != "" {
		id = url.PathEscape(strings.Trim(repo, "/"))
	}
	return "projects/" + id + "/" + endpoint
}

func listArgs(args []string) (int, []string, []string, error) {
	limit, fields, query := 30, []string(nil), []string{}
	for i := 0; i < len(args); i++ {
		key, value, consumed, err := flagValue(args, i)
		if err != nil {
			return 0, nil, nil, err
		}
		if consumed {
			i++
		}
		switch key {
		case "--limit":
			limit, err = strconv.Atoi(value)
			if err != nil || limit < 1 || limit > 100 {
				return 0, nil, nil, errors.New("--limit must be between 1 and 100")
			}
		case "--fields":
			fields = strings.Split(value, ",")
		case "--state", "--scope", "--status", "--ref", "--author", "--assignee", "--labels", "--search":
			query = append(query, url.QueryEscape(strings.TrimPrefix(key, "--"))+"="+url.QueryEscape(value))
		default:
			return 0, nil, nil, errors.New("unsupported flag: " + key)
		}
	}
	return limit, fields, query, nil
}

func flagValue(args []string, i int) (string, string, bool, error) {
	if !strings.HasPrefix(args[i], "--") {
		return args[i], "", false, errors.New("unexpected argument: " + args[i])
	}
	if p := strings.IndexByte(args[i], '='); p >= 0 {
		return args[i][:p], args[i][p+1:], false, nil
	}
	if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
		return args[i], "", false, errors.New(args[i] + " requires a value")
	}
	return args[i], args[i+1], true, nil
}

func takeValue(args []string, names ...string) (string, []string, error) {
	for i, arg := range args {
		for _, name := range names {
			if arg == name {
				if i+1 >= len(args) {
					return "", args, errors.New(name + " requires a value")
				}
				return args[i+1], append(args[:i], args[i+2:]...), nil
			}
			if strings.HasPrefix(arg, name+"=") {
				return strings.TrimPrefix(arg, name+"="), append(args[:i], args[i+1:]...), nil
			}
		}
	}
	return "", args, nil
}

func help(args []string) string {
	if len(args) > 0 {
		if args[0] == "api" {
			return "usage: glab-axi api <endpoint> [glab-api-flags]\nexamples[3]:\n  glab-axi api user\n  glab-axi api projects/:id/releases\n  glab-axi api -X POST projects/:id/trigger/pipeline -f ref=main\n"
		}
		if r, ok := resources[args[0]]; ok {
			return fmt.Sprintf("usage: glab-axi %s <list|view> [flags]\nflags[5]:\n  --state <state>, --limit <n> (default 30, max 100), --fields <a,b>, -R/--repo <group/project>, --full (view)\nexamples[3]:\n  glab-axi %s list\n  glab-axi %s list --state opened --limit 50\n  glab-axi %s view <id> --full\n", commandName(r), commandName(r), commandName(r), commandName(r))
		}
	}
	return "usage: glab-axi [command] [flags]\ncommands[7]:\n  (none)=dashboard, issue, mr, pipeline, label, api, help\nflags[3]:\n  -R/--repo <group/project>, --help, -v/-V/--version\nexamples[4]:\n  glab-axi\n  glab-axi issue list --state opened\n  glab-axi mr view 42\n  glab-axi pipeline list --status failed\n"
}

func has(args []string, value string) bool {
	for _, arg := range args {
		if arg == value {
			return true
		}
	}
	return false
}
func singular(s string) string { return strings.TrimSuffix(s, "s") }
func commandName(r resource) string {
	if r.name == "merge_requests" {
		return "mr"
	}
	return singular(r.name)
}
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
func usageError(message, suggestion string) int {
	fmt.Printf("error: %s\nhelp: %s\n", quote(message), quote(suggestion))
	return 2
}
func commandError(err error) int {
	fmt.Printf("error: %s\nhelp: Run `glab-axi api user` to verify authentication, then check the project and arguments\n", quote(err.Error()))
	return 1
}
