package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var (
	url      string
	gitDir   string
	workTree string
	sshKey   string
	debug    bool
)

type InitCmd struct {
	log   Logger
	flags *flag.FlagSet
}

func NewInitCmd(log Logger) InitCmd {
	flags := flag.NewFlagSet("init", flag.ExitOnError)
	flags.StringVar(&url, "url", "", "The git remote repository URL that stores your system configuration files.")
	flags.StringVar(&gitDir, "git.dir", "~/.dotfiles", "The git bare directory location.")
	flags.StringVar(&workTree, "work.tree", "~/", "All system config files should be discoverable within this root directory.")
	flags.StringVar(&sshKey, "ssh.key", "~/.ssh/id_rsa", "The ssh key used to interact with your git repository storing your configuration files.")
	flags.BoolVar(&debug, "debug", false, "Output any errors in full during initialisation.")
	return InitCmd{
		log:   log,
		flags: flags,
	}
}

func (i InitCmd) Run(args ...string) int {
	err := i.flags.Parse(args)
	if err != nil {
		if debug {
			i.log.Println(err)
		}
		return 1
	}

	conf := CLIConfig{
		GitDir:   gitDir,
		WorkTree: workTree,
	}
	_, err = UpdateConfig(conf)
	if err != nil {
		if debug {
			i.log.Println(err)
		}
		return 1
	}
	if err := runGit(os.Stdout, "init", "--bare", gitDir); err != nil {
		if debug {
			i.log.Println(err)
		}
		return 1
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	if err := runGit(buf, "--git-dir", gitDir, "remote", "show"); err != nil {
		if debug {
			i.log.Println(err)
		}
		return 1
	}

	// We are re-initialising the repo, the origin might already be set
	if strings.TrimSpace(buf.String()) != "origin" {
		if err := runGit(os.Stdout, "--git-dir", gitDir, "remote", "add", "origin", url); err != nil {
			if debug {
				i.log.Println(err)
			}
			return 1
		}
	} else {
		if err := runGit(os.Stdout, "--git-dir", gitDir, "remote", "set-url", "origin", url); err != nil {
			if debug {
				i.log.Println(err)
			}
			return 1
		}
	}

	if err := runGit(os.Stdout, "--git-dir", gitDir, "config", "--local", "status.showUntrackedFiles", "no"); err != nil {
		if debug {
			i.log.Println(err)
		}
		return 1
	}
	if err := runGit(os.Stdout, "--git-dir", gitDir, "config", "--local", "core.sshCommand", fmt.Sprintf("ssh -i %s", sshKey)); err != nil {
		if debug {
			i.log.Println(err)
		}
		return 1
	}
	return 0
}

func (i InitCmd) Help() string {
	buf := bytes.NewBuffer(make([]byte, 0))
	i.flags.SetOutput(buf)
	i.flags.PrintDefaults()
	return buf.String()
}

func runGit(out io.Writer, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
