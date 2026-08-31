package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/hkx5414375/scaffold-agent/internal/engine"
	"github.com/hkx5414375/scaffold-agent/internal/plan"
	"github.com/hkx5414375/scaffold-agent/internal/result"
	"github.com/hkx5414375/scaffold-agent/internal/version"
)

func runEngineCommand(command string, args []string, stdout, stderr io.Writer) int {
	application := engine.New(version.Current().Version)
	ctx := context.Background()
	var envelope result.Envelope
	var err error
	switch command {
	case "query":
		envelope, err = runQuery(ctx, application, args, stderr)
	case "validate":
		envelope, err = runValidate(ctx, application, args, stderr)
	case "plan":
		envelope, err = runPlan(ctx, application, args, stderr)
	case "preview":
		envelope, err = runPreview(ctx, application, args, stderr)
	case "apply":
		envelope, err = runApply(ctx, application, args, stderr)
	case "verify":
		envelope, err = runVerify(ctx, application, args, stderr)
	case "result":
		envelope, err = runResult(ctx, application, args, stderr)
	case "rollback":
		envelope, err = runTransaction(ctx, application.Rollback, "rollback", args, stderr)
	case "recover":
		envelope, err = runTransaction(ctx, application.Recover, "recover", args, stderr)
	default:
		return 2
	}
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	return writeEnvelope(stdout, stderr, envelope)
}

func runQuery(ctx context.Context, application *engine.Engine, args []string, stderr io.Writer) (result.Envelope, error) {
	flags := newFlagSet("query", stderr)
	topic := flags.String("topic", "support", "support, workflow, or project")
	root := flags.String("project-root", "", "absolute project root")
	if err := parseFlags(flags, args); err != nil {
		return result.Envelope{}, err
	}
	return application.Query(ctx, engine.QueryInput{Topic: *topic, ProjectRoot: *root}), nil
}

func runValidate(ctx context.Context, application *engine.Engine, args []string, stderr io.Writer) (result.Envelope, error) {
	flags := newFlagSet("validate", stderr)
	root := flags.String("project-root", "", "absolute project root")
	blueprint := flags.String("blueprint", "scaffold.yaml", "project-relative Blueprint path")
	if err := parseFlags(flags, args); err != nil {
		return result.Envelope{}, err
	}
	return application.Validate(ctx, engine.ValidateInput{ProjectRoot: *root, BlueprintPath: *blueprint}), nil
}

func runPlan(ctx context.Context, application *engine.Engine, args []string, stderr io.Writer) (result.Envelope, error) {
	flags := newFlagSet("plan", stderr)
	root := flags.String("project-root", "", "absolute project root")
	blueprint := flags.String("blueprint", "scaffold.yaml", "project-relative Blueprint path")
	action := flags.String("action", "create", "create, modify, extend, reduce, repair, or upgrade")
	if err := parseFlags(flags, args); err != nil {
		return result.Envelope{}, err
	}
	return application.Plan(ctx, engine.PlanInput{ProjectRoot: *root, BlueprintPath: *blueprint, Action: plan.Action(*action)}), nil
}

func runPreview(ctx context.Context, application *engine.Engine, args []string, stderr io.Writer) (result.Envelope, error) {
	flags := newFlagSet("preview", stderr)
	root := flags.String("project-root", "", "absolute project root")
	planID := flags.String("plan-id", "", "content-addressed plan ID")
	cursor := flags.String("cursor", "", "opaque page cursor")
	limit := flags.Int("limit", 0, "page size from 1 to 100")
	if err := parseFlags(flags, args); err != nil {
		return result.Envelope{}, err
	}
	return application.Preview(ctx, engine.PreviewInput{ProjectRoot: *root, PlanID: *planID, Cursor: *cursor, Limit: *limit}), nil
}

func runApply(ctx context.Context, application *engine.Engine, args []string, stderr io.Writer) (result.Envelope, error) {
	flags := newFlagSet("apply", stderr)
	root := flags.String("project-root", "", "absolute project root")
	planID := flags.String("plan-id", "", "content-addressed plan ID")
	applyToken := flags.String("apply-token", "", "token returned by preview")
	if err := parseFlags(flags, args); err != nil {
		return result.Envelope{}, err
	}
	return application.Apply(ctx, engine.ApplyInput{ProjectRoot: *root, PlanID: *planID, ApplyToken: *applyToken}), nil
}

func runVerify(ctx context.Context, application *engine.Engine, args []string, stderr io.Writer) (result.Envelope, error) {
	flags := newFlagSet("verify", stderr)
	root := flags.String("project-root", "", "absolute project root")
	limit := flags.Int("limit", 0, "first-page size from 1 to 100")
	if err := parseFlags(flags, args); err != nil {
		return result.Envelope{}, err
	}
	return application.Verify(ctx, engine.VerifyInput{ProjectRoot: *root, Limit: *limit}), nil
}

func runResult(ctx context.Context, application *engine.Engine, args []string, stderr io.Writer) (result.Envelope, error) {
	flags := newFlagSet("result", stderr)
	root := flags.String("project-root", "", "absolute project root")
	resultID := flags.String("result-id", "", "content-addressed result ID")
	cursor := flags.String("cursor", "", "opaque page cursor")
	limit := flags.Int("limit", 0, "page size from 1 to 100")
	if err := parseFlags(flags, args); err != nil {
		return result.Envelope{}, err
	}
	return application.Result(ctx, engine.ResultInput{ProjectRoot: *root, ResultID: *resultID, Cursor: *cursor, Limit: *limit}), nil
}

func runTransaction(ctx context.Context, operation func(context.Context, engine.TransactionInput) result.Envelope, name string, args []string, stderr io.Writer) (result.Envelope, error) {
	flags := newFlagSet(name, stderr)
	root := flags.String("project-root", "", "absolute project root")
	planID := flags.String("plan-id", "", "content-addressed plan ID")
	if err := parseFlags(flags, args); err != nil {
		return result.Envelope{}, err
	}
	return operation(ctx, engine.TransactionInput{ProjectRoot: *root, PlanID: *planID}), nil
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		_, _ = fmt.Fprintf(stderr, "Usage of scaffold-agent %s:\n", name)
		flags.PrintDefaults()
	}
	return flags
}

func parseFlags(flags *flag.FlagSet, args []string) error {
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintf(flags.Output(), "unexpected positional arguments: %v\n", flags.Args())
		return errors.New("unexpected positional arguments")
	}
	return nil
}
