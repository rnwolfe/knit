package cli

import (
	"github.com/rnwolfe/knit/internal/api"
	"github.com/rnwolfe/knit/internal/errs"
)

// ReplyCmd groups reply reads + reply management. Reply text comes from other users → fenced
// in agent mode (contract §8). hide/unhide are idempotent (contract §9).
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
	replies, cursor, err := rt.API.ListReplies(rt.Ctx, c.ID, api.PageOpts{Limit: rt.Cfg.Limit, Cursor: c.Cursor})
	if err != nil {
		return err
	}
	rt.fenceReplies(replies)
	return rt.Out.EmitEnvelope(replies, cursor)
}

type ReplyTreeCmd struct {
	ID     string `arg:"" help:"Media id of the post."`
	Cursor string `help:"Opaque pagination cursor."`
}

func (c *ReplyTreeCmd) Run(rt *Runtime) error {
	replies, cursor, err := rt.API.Conversation(rt.Ctx, c.ID, api.PageOpts{Limit: rt.Cfg.Limit, Cursor: c.Cursor})
	if err != nil {
		return err
	}
	rt.fenceReplies(replies)
	return rt.Out.EmitEnvelope(replies, cursor)
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
	var images []string
	if c.Image != "" {
		images = []string{c.Image}
	}
	p, err := rt.API.Publish(rt.Ctx, api.PublishReq{Text: c.Text, ImageURLs: images, ReplyToID: c.ID})
	if err != nil {
		return err
	}
	return rt.Out.Emit(map[string]any{"ok": true, "id": p.ID, "permalink": deref(p.Permalink)})
}

// --- reply hide / unhide (idempotent mutations) -----------------------------

type ReplyHideCmd struct {
	ID string `arg:"" help:"Reply id to hide."`
}

func (c *ReplyHideCmd) Run(rt *Runtime) error { return manageReply(rt, c.ID, true, "reply hide") }

type ReplyUnhideCmd struct {
	ID string `arg:"" help:"Reply id to unhide."`
}

func (c *ReplyUnhideCmd) Run(rt *Runtime) error { return manageReply(rt, c.ID, false, "reply unhide") }

func manageReply(rt *Runtime, id string, hide bool, op string) error {
	if err := rt.Guard(op); err != nil {
		return err
	}
	if rt.Cfg.DryRun {
		return rt.Out.Emit(map[string]any{"dryRun": true, "action": op, "id": id})
	}
	r, err := rt.API.ManageReply(rt.Ctx, id, hide)
	if err != nil {
		return err
	}
	return rt.Out.Emit(map[string]any{"ok": true, "id": r.ID, "hideStatus": r.HideStatus})
}
