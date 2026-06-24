package cli

import "github.com/rnwolfe/knit/internal/errs"

// ReplyCmd groups reply reads + reply management. Reads return free text from other users
// → fence as untrusted in agent mode (contract §8). hide/unhide are idempotent (contract §9).
type ReplyCmd struct {
	List   ReplyListCmd   `cmd:"" help:"List top-level replies to a post."`
	Tree   ReplyTreeCmd   `cmd:"" help:"Show the full conversation tree for a post."`
	Create ReplyCreateCmd `cmd:"" help:"Reply to a post (mutation)."`
	Hide   ReplyHideCmd   `cmd:"" help:"Hide a reply on your post (mutation, idempotent)."`
	Unhide ReplyUnhideCmd `cmd:"" help:"Unhide a reply on your post (mutation, idempotent)."`
}

// --- reply list / tree (reads) ----------------------------------------------

type ReplyListCmd struct {
	ID     string `arg:"" help:"Media id of the post."`
	Cursor string `help:"Opaque pagination cursor."`
}

func (c *ReplyListCmd) Run(rt *Runtime) error {
	// PLACEHOLDER: cli-implement wires `GET /{media-id}/replies` and fences the text.
	return rt.Out.EmitEnvelope([]any{}, "")
}

type ReplyTreeCmd struct {
	ID     string `arg:"" help:"Media id of the post."`
	Cursor string `help:"Opaque pagination cursor."`
}

func (c *ReplyTreeCmd) Run(rt *Runtime) error {
	// PLACEHOLDER: cli-implement wires `GET /{media-id}/conversation`.
	return rt.Out.EmitEnvelope([]any{}, "")
}

// --- reply create (mutation) ------------------------------------------------

type ReplyCreateCmd struct {
	ID    string `arg:"" help:"Media id to reply to."`
	Text  string `help:"Reply text."`
	Image string `help:"Image URL."`
	Apply string `help:"Publish only the plan whose --dry-run hash equals this value."`
}

func (c *ReplyCreateCmd) Run(rt *Runtime) error {
	if err := rt.Guard("reply create"); err != nil {
		return err
	}
	if c.Text == "" && c.Image == "" {
		if rt.Cfg.NoInput {
			return errs.InputRequired("text or media")
		}
		return errs.New(errs.ExitUsage, "USAGE", "a reply needs --text and/or --image",
			`knit reply create <media-id> --text "..." --allow-mutations`)
	}
	plan := map[string]any{"action": "reply.create", "id": c.ID, "text": c.Text, "image": c.Image}
	hash := planHash(plan)
	if rt.Cfg.DryRun {
		return rt.Out.Emit(map[string]any{"dryRun": true, "plan": plan, "hash": hash})
	}
	if c.Apply != "" && c.Apply != hash {
		return errs.New(errs.ExitUsage, "PLAN_MISMATCH", "the --apply hash does not match the current plan",
			"re-run with --dry-run to get the current hash")
	}
	// PLACEHOLDER publish.
	return rt.Out.Emit(map[string]any{"ok": true, "id": "placeholder", "permalink": ""})
}

// --- reply hide / unhide (idempotent mutations) -----------------------------

type ReplyHideCmd struct {
	ID string `arg:"" help:"Reply id to hide."`
}

func (c *ReplyHideCmd) Run(rt *Runtime) error { return setHidden(rt, c.ID, true, "reply hide") }

type ReplyUnhideCmd struct {
	ID string `arg:"" help:"Reply id to unhide."`
}

func (c *ReplyUnhideCmd) Run(rt *Runtime) error { return setHidden(rt, c.ID, false, "reply unhide") }

func setHidden(rt *Runtime, id string, hidden bool, op string) error {
	if err := rt.Guard(op); err != nil {
		return err
	}
	if rt.Cfg.DryRun {
		return rt.Out.Emit(map[string]any{"dryRun": true, "action": op, "id": id})
	}
	// Idempotent: setting the state it already has is a soft success (contract §9).
	changed, err := rt.Store.SetReplyHidden(id, hidden)
	if err != nil {
		return errs.New(errs.ExitConfig, "STORE_ERROR", err.Error(), "check the store path")
	}
	status := "NOT_HUSHED"
	if hidden {
		status = "HIDDEN"
	}
	return rt.Out.Emit(map[string]any{"ok": true, "id": id, "hideStatus": status, "changed": changed})
}
