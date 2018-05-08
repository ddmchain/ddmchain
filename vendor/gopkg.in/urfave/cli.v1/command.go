package cli

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
)

type Command struct {

	Name string

	ShortName string

	Aliases []string

	Usage string

	UsageText string

	Description string

	ArgsUsage string

	Category string

	BashComplete BashCompleteFunc

	Before BeforeFunc

	After AfterFunc

	Action interface{}

	OnUsageError OnUsageErrorFunc

	Subcommands Commands

	Flags []Flag

	SkipFlagParsing bool

	SkipArgReorder bool

	HideHelp bool

	Hidden bool

	HelpName        string
	commandNamePath []string

	CustomHelpTemplate string
}

type CommandsByName []Command

func (c CommandsByName) Len() int {
	return len(c)
}

func (c CommandsByName) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

func (c CommandsByName) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c Command) FullName() string {
	if c.commandNamePath == nil {
		return c.Name
	}
	return strings.Join(c.commandNamePath, " ")
}

type Commands []Command

func (c Command) Run(ctx *Context) (err error) {
	if len(c.Subcommands) > 0 {
		return c.startApp(ctx)
	}

	if !c.HideHelp && (HelpFlag != BoolFlag{}) {

		c.Flags = append(
			c.Flags,
			HelpFlag,
		)
	}

	set, err := flagSet(c.Name, c.Flags)
	if err != nil {
		return err
	}
	set.SetOutput(ioutil.Discard)

	if c.SkipFlagParsing {
		err = set.Parse(append([]string{"--"}, ctx.Args().Tail()...))
	} else if !c.SkipArgReorder {
		firstFlagIndex := -1
		terminatorIndex := -1
		for index, arg := range ctx.Args() {
			if arg == "--" {
				terminatorIndex = index
				break
			} else if arg == "-" {

				continue
			} else if strings.HasPrefix(arg, "-") && firstFlagIndex == -1 {
				firstFlagIndex = index
			}
		}

		if firstFlagIndex > -1 {
			args := ctx.Args()
			regularArgs := make([]string, len(args[1:firstFlagIndex]))
			copy(regularArgs, args[1:firstFlagIndex])

			var flagArgs []string
			if terminatorIndex > -1 {
				flagArgs = args[firstFlagIndex:terminatorIndex]
				regularArgs = append(regularArgs, args[terminatorIndex:]...)
			} else {
				flagArgs = args[firstFlagIndex:]
			}

			err = set.Parse(append(flagArgs, regularArgs...))
		} else {
			err = set.Parse(ctx.Args().Tail())
		}
	} else {
		err = set.Parse(ctx.Args().Tail())
	}

	nerr := normalizeFlags(c.Flags, set)
	if nerr != nil {
		fmt.Fprintln(ctx.App.Writer, nerr)
		fmt.Fprintln(ctx.App.Writer)
		ShowCommandHelp(ctx, c.Name)
		return nerr
	}

	context := NewContext(ctx.App, set, ctx)
	context.Command = c
	if checkCommandCompletions(context, c.Name) {
		return nil
	}

	if err != nil {
		if c.OnUsageError != nil {
			err := c.OnUsageError(context, err, false)
			HandleExitCoder(err)
			return err
		}
		fmt.Fprintln(context.App.Writer, "Incorrect Usage:", err.Error())
		fmt.Fprintln(context.App.Writer)
		ShowCommandHelp(context, c.Name)
		return err
	}

	if checkCommandHelp(context, c.Name) {
		return nil
	}

	if c.After != nil {
		defer func() {
			afterErr := c.After(context)
			if afterErr != nil {
				HandleExitCoder(err)
				if err != nil {
					err = NewMultiError(err, afterErr)
				} else {
					err = afterErr
				}
			}
		}()
	}

	if c.Before != nil {
		err = c.Before(context)
		if err != nil {
			ShowCommandHelp(context, c.Name)
			HandleExitCoder(err)
			return err
		}
	}

	if c.Action == nil {
		c.Action = helpSubcommand.Action
	}

	err = HandleAction(c.Action, context)

	if err != nil {
		HandleExitCoder(err)
	}
	return err
}

func (c Command) Names() []string {
	names := []string{c.Name}

	if c.ShortName != "" {
		names = append(names, c.ShortName)
	}

	return append(names, c.Aliases...)
}

func (c Command) HasName(name string) bool {
	for _, n := range c.Names() {
		if n == name {
			return true
		}
	}
	return false
}

func (c Command) startApp(ctx *Context) error {
	app := NewApp()
	app.Metadata = ctx.App.Metadata

	app.Name = fmt.Sprintf("%s %s", ctx.App.Name, c.Name)
	if c.HelpName == "" {
		app.HelpName = c.HelpName
	} else {
		app.HelpName = app.Name
	}

	app.Usage = c.Usage
	app.Description = c.Description
	app.ArgsUsage = c.ArgsUsage

	app.CommandNotFound = ctx.App.CommandNotFound
	app.CustomAppHelpTemplate = c.CustomHelpTemplate

	app.Commands = c.Subcommands
	app.Flags = c.Flags
	app.HideHelp = c.HideHelp

	app.Version = ctx.App.Version
	app.HideVersion = ctx.App.HideVersion
	app.Compiled = ctx.App.Compiled
	app.Author = ctx.App.Author
	app.Email = ctx.App.Email
	app.Writer = ctx.App.Writer
	app.ErrWriter = ctx.App.ErrWriter

	app.categories = CommandCategories{}
	for _, command := range c.Subcommands {
		app.categories = app.categories.AddCommand(command.Category, command)
	}

	sort.Sort(app.categories)

	app.EnableBashCompletion = ctx.App.EnableBashCompletion
	if c.BashComplete != nil {
		app.BashComplete = c.BashComplete
	}

	app.Before = c.Before
	app.After = c.After
	if c.Action != nil {
		app.Action = c.Action
	} else {
		app.Action = helpSubcommand.Action
	}
	app.OnUsageError = c.OnUsageError

	for index, cc := range app.Commands {
		app.Commands[index].commandNamePath = []string{c.Name, cc.Name}
	}

	return app.RunAsSubcommand(ctx)
}

func (c Command) VisibleFlags() []Flag {
	return visibleFlags(c.Flags)
}
