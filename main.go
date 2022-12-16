// Copyright 2022 c-fraser
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-git/go-git/v5"
	"github.com/urfave/cli/v2"
)

// main is the entry point into the jx application.
func main() {
	executable("java")
	var conf config
	conf.read()
	app := cli.NewApp()
	app.Name = "jx"
	app.Usage = "JVM application executor"
	app.Version = "0.2.0"
	app.Commands = cli.Commands{
		&cli.Command{
			Name:  "install",
			Usage: "Install a JVM application",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "git",
					Usage:    "The `URL` of the project's git repository",
					Required: true,
				},
				&cli.StringFlag{
					Name:        "name",
					Usage:       "The `NAME` of the project",
					DefaultText: "the name of the project's git repository",
					Required:    false,
				},
				&cli.StringFlag{
					Name:        "build",
					Usage:       "The `COMMAND` to build the project",
					DefaultText: "./gradlew installDist",
					Required:    false,
				},
				&cli.StringFlag{
					Name:        "execute",
					Usage:       "The `COMMAND` to execute the project",
					DefaultText: "./build/install/$project/bin/$name $args",
					Required:    false,
				},
			},
			Action: func(ctx *cli.Context) error {
				url := ctx.String("git")
				name := ctx.String("name")
				base := path.Base(url)
				directory := filepath.Join(
					filepath.Dir(conf.File),
					base[:len(base)-len(filepath.Ext(base))],
				)
				if name == "" {
					name = path.Base(directory)
				}
				build := ctx.String("build")
				command := ctx.String("execute")
				return display(
					"Installing...",
					fmt.Sprintf("🚀 Installed %s!", name),
					func() error {
						return install(&conf, url, directory, name, build, command)
					},
				)
			},
		},
		&cli.Command{
			Name:      "run",
			Usage:     "Execute an installed JVM application",
			ArgsUsage: "[name of installed project to execute]",
			Action: func(ctx *cli.Context) error {
				return run(&conf, ctx.Args().Slice()...)
			},
		},
		&cli.Command{
			Name:      "upgrade",
			Usage:     "Upgrade installed JVM application(s)",
			ArgsUsage: "[name of projects to upgrade]",
			Action: func(ctx *cli.Context) error {
				names := ctx.Args().Slice()
				return display(
					"Upgrading...",
					fmt.Sprintf("🛠 Upgraded %s!", strings.Join(names, ", ")),
					func() error {
						return upgrade(&conf, names...)
					},
				)
			},
		},
		&cli.Command{
			Name:      "uninstall",
			Usage:     "Uninstall JVM application(s)",
			ArgsUsage: "[name of projects to uninstall]",
			Action: func(ctx *cli.Context) error {
				names := ctx.Args().Slice()
				return display(
					"Uninstalling...",
					fmt.Sprintf("✨ Uninstalled %s!", strings.Join(names, ", ")),
					func() error {
						return uninstall(&conf, names...)
					},
				)
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		exit(err)
	}
	conf.write()
}

// executable verifies that the command is accessible via the path.
func executable(command string) {
	if runtime.GOOS == "windows" {
		command = command + ".exe"
	}
	_, err := exec.LookPath(command)
	if errors.Is(err, exec.ErrDot) {
		err = nil
	}
	if err != nil {
		exit(fmt.Errorf("❌ %s is required: %w", command, err))
	}
}

// config stores the jx application configuration.
type config struct {
	// The File where the configuration is stored as JSON.
	File string `json:"-"`
	// The installed Projects, accessible by name.
	Projects map[string]project `json:"projects"`
}

// project is an installed and executable JVM application.
type project struct {
	// The Name of the project.
	Name string `json:"-"`
	// The path of the project's local git repository.
	Repository string `json:"repository"`
	// The Url of the project's git repository.
	Url string `json:"url"`
	// The git Reference of the project.
	Reference [20]byte `json:"reference"`
	// The command to Build the project.
	Build string `json:"build"`
	// The command to Execute the project.
	Execute string `json:"execute"`
}

// read the config from the 'config.json' File, if it exists.
func (c *config) read() {
	home, err := os.UserHomeDir()
	if err != nil {
		exit(fmt.Errorf("❌ unable to find home directory: %w", err))
	}
	directory := filepath.Join(home, ".jx")
	err = os.MkdirAll(directory, os.ModePerm)
	if err != nil {
		exit(fmt.Errorf("❌ failed to create application directory: %w", err))
	}
	c.File = path.Join(directory, "config.json")
	_, err = os.Stat(c.File)
	if err != nil {
		c.Projects = make(map[string]project)
		return
	}
	bytes, err := os.ReadFile(c.File)
	if err != nil {
		exit(fmt.Errorf("❌ failed to read config file: %w", err))
	}
	err = json.Unmarshal(bytes, c)
	if err != nil {
		exit(fmt.Errorf("❌ failed to parse config: %w", err))
	}
}

// write the config to the 'config.json' File, or delete the jx configuration directory if no
// config.Projects are installed.
func (c *config) write() {
	if len(c.Projects) == 0 {
		_ = os.RemoveAll(filepath.Dir(c.File))
		return
	}
	bytes, err := json.Marshal(*c)
	if err != nil {
		exit(fmt.Errorf("❌ failed to encode config: %w", err))
	}
	err = os.WriteFile(c.File, bytes, os.ModePerm)
	if err != nil {
		exit(fmt.Errorf("❌ failed to write config file %w", err))
	}
}

// install the project at the url with the name, then build the project and store it in the config.
func install(conf *config, url, directory, name, build, command string) error {
	if _, ok := conf.Projects[name]; ok {
		return fmt.Errorf("❌ %s is already installed", url)
	}
	repository, err := clone(directory, url)
	if err != nil {
		return err
	}
	reference, err := repository.Head()
	if err != nil {
		_ = os.RemoveAll(directory)
		return err
	}
	if build == "" {
		gradlew := "gradlew"
		if runtime.GOOS == "windows" {
			gradlew = gradlew + ".bat"
		}
		build = filepath.Join(directory, gradlew) + " installDist"
	}
	err = execute(directory, build, false)
	if err != nil {
		_ = os.RemoveAll(directory)
		return err
	}
	if command == "" {
		base := filepath.Join(directory, "build", "install", name, "bin")
		file := name
		if runtime.GOOS == "windows" {
			file = file + ".exe"
		}
		command = filepath.Join(base, file)
	}
	conf.Projects[name] = project{
		Name:       name,
		Repository: directory,
		Url:        url,
		Reference:  reference.Hash(),
		Build:      build,
		Execute:    command,
	}
	return nil
}

// clone the git repository at the url into the directory.
func clone(directory, url string) (*git.Repository, error) {
	repository, err := git.PlainClone(directory, false, &git.CloneOptions{URL: url})
	if err != nil {
		return nil, err
	}
	return repository, nil
}

// run the project with the name.
func run(conf *config, command ...string) error {
	if len(command) == 0 {
		return errors.New("❌ project name is required")
	}
	name := command[0]
	target, ok := conf.Projects[name]
	if !ok {
		return fmt.Errorf("❌ %s is not installed", name)
	}
	if _, err := os.Stat(target.Repository); os.IsNotExist(err) {
		return fmt.Errorf("❌ %s install is invalid", name)
	}
	command[0] = target.Execute
	_ = execute(target.Repository, strings.Join(command, " "), true)
	return nil
}

// upgrade the installed config.Projects with the names.
func upgrade(conf *config, names ...string) error {
	if len(names) == 0 {
		return errors.New("❌ project name is required")
	}
	projects := make([]project, 0)
	if len(names) > 0 {
		for _, name := range names {
			if p, ok := conf.Projects[name]; ok {
				p.Name = name
				projects = append(projects, p)
			} else {
				return fmt.Errorf("❌ %s is not installed", name)
			}
		}
	} else {
		for n, p := range conf.Projects {
			p.Name = n
			projects = append(projects, p)
		}
	}
	for i := range projects {
		p := projects[i]
		repository, err := open(p.Repository)
		if err != nil {
			return err
		}
		err = repository.Fetch(&git.FetchOptions{})
		if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return err
		}
		err = execute(p.Repository, p.Build, false)
		if err != nil {
			return err
		}
		reference, err := repository.Head()
		if err != nil {
			continue
		}
		p.Reference = reference.Hash()
		conf.Projects[p.Name] = p
	}
	return nil
}

// open the git repository in the directory.
func open(directory string) (*git.Repository, error) {
	repository, err := git.PlainOpen(directory)
	if err != nil {
		return nil, err
	}
	return repository, nil
}

// uninstall the config.Projects with the names.
func uninstall(conf *config, names ...string) error {
	if len(names) == 0 {
		return errors.New("❌ project name is required")
	}
	projects := make([]project, 0)
	if len(names) > 0 {
		for _, name := range names {
			if p, ok := conf.Projects[name]; ok {
				p.Name = name
				projects = append(projects, p)
			} else {
				return fmt.Errorf("❌ %s is not installed", name)
			}
		}
	} else {
		for name, p := range conf.Projects {
			p.Name = name
			projects = append(projects, p)
		}
	}
	for _, p := range projects {
		err := os.RemoveAll(p.Repository)
		if err != nil {
			return err
		}
		delete(conf.Projects, p.Name)
	}
	return nil
}

// execute the command, optionally interactively, in the directory.
func execute(directory, command string, interactive bool) error {
	args := strings.Fields(command)
	var cmd *exec.Cmd
	switch len(args) {
	case 0:
		return errors.New("❌ empty command")
	case 1:
		cmd = exec.Command(args[0])
	default:
		cmd = exec.Command(args[0], args[1:]...)
	}
	cmd.Dir = directory
	if interactive {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// exit the application after printing the error.
func exit(err error) {
	fmt.Println(err)
	os.Exit(1)
}

// display the running and completed messages when executing the command.
func display(running, completed string, command func() error) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	program := tea.NewProgram(model{spinner: s, running: running, completed: completed})
	go func() {
		err := command()
		if err != nil {
			program.Send(err)
		} else {
			program.Send(success{})
		}
	}()
	i, err := program.Run()
	if err != nil {
		return err
	}
	if m, ok := i.(model); ok && m.err != nil {
		return m.err
	}
	return nil
}

// success is an internal message to signal the successful completion of a CLI command.
type success struct{}

// model is the tea.Model implementation which is executed in display.
type model struct {
	spinner   spinner.Model
	running   string
	completed string
	quit      bool
	err       error
}

// Init the model with the spinner.Tick command.
func (m model) Init() tea.Cmd {
	return spinner.Tick
}

// Update the model with the message.
func (m model) Update(i tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := i.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		default:
			return m, nil
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case error:
		m.err = msg
		m.quit = true
		return m, tea.Quit
	case success:
		m.quit = true
		return m, tea.Quit
	default:
		return m, nil
	}
}

// View the model. Displays the spinner and messages.
func (m model) View() string {
	if m.quit && m.err == nil {
		return m.completed + "\n"
	} else {
		return m.spinner.View() + " " + m.running
	}
}
