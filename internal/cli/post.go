package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/rnwolfe/knit/internal/api"
	"github.com/rnwolfe/knit/internal/errs"
)

// PostCmd groups the post noun-verb subcommands (the primary noun).
type PostCmd struct {
	List   PostListCmd   `cmd:"" help:"List the authenticated user's posts."`
	Get    PostGetCmd    `cmd:"" help:"Get one post by media id."`
	Create PostCreateCmd `cmd:"" help:"Publish a post (mutation; reviewed-artifact via --dry-run/--apply)."`
	Repost PostRepostCmd `cmd:"" help:"Repost a post by media id (mutation)."`
	Delete PostDeleteCmd `cmd:"" help:"Delete a post by media id (mutation, idempotent)."`
}

// --- post list --------------------------------------------------------------

type PostListCmd struct {
	Since  string `help:"Only posts on/after this ISO-8601 date."`
	Until  string `help:"Only posts on/before this ISO-8601 date."`
	Cursor string `help:"Opaque pagination cursor from a previous nextCursor."`
}

func (c *PostListCmd) Run(rt *Runtime) error {
	posts, cursor, err := rt.API.ListPosts(rt.Ctx, api.PageOpts{
		Limit: rt.Cfg.Limit, Cursor: c.Cursor, Since: c.Since, Until: c.Until,
	})
	if err != nil {
		return err
	}
	rt.fencePosts(posts)
	return rt.Out.EmitEnvelope(posts, cursor)
}

// --- post get ---------------------------------------------------------------

type PostGetCmd struct {
	ID string `arg:"" help:"Media id of the post."`
}

func (c *PostGetCmd) Run(rt *Runtime) error {
	p, err := rt.API.GetPost(rt.Ctx, c.ID)
	if err != nil {
		return err
	}
	rt.fencePost(p)
	return rt.Out.EmitEnvelope(p, "")
}

// --- post create ------------------------------------------------------------

// PostCreateCmd publishes a post. High-stakes + irreversible + public, so it uses the
// reviewed-artifact = approval pattern (contract §2): `--dry-run` emits the exact plan plus a
// content hash; `--apply <hash>` publishes only if the recomputed hash matches.
type PostCreateCmd struct {
	Text          string   `help:"Post text."`
	Image         []string `help:"Image URL (repeatable for a carousel)."`
	Video         []string `help:"Video URL (repeatable for a carousel)."`
	Link          string   `help:"Link attachment URL."`
	ReplyTo       string   `help:"Reply to this media id."`
	Quote         string   `help:"Quote this media id."`
	Topic         string   `help:"Topic tag."`
	ReplyAudience string   `enum:",everyone,accounts_you_follow,mentioned_only" default:"" help:"Who can reply."`
	Apply         string   `help:"Publish only the plan whose --dry-run hash equals this value."`
}

func (c *PostCreateCmd) plan() map[string]any {
	return map[string]any{
		"action":        "post.create",
		"text":          c.Text,
		"image":         c.Image,
		"video":         c.Video,
		"link":          c.Link,
		"replyTo":       c.ReplyTo,
		"quote":         c.Quote,
		"topic":         c.Topic,
		"replyAudience": c.ReplyAudience,
	}
}

func (c *PostCreateCmd) Run(rt *Runtime) error {
	if err := rt.Guard("post create"); err != nil {
		return err
	}
	if c.Text == "" && len(c.Image) == 0 && len(c.Video) == 0 {
		if rt.Cfg.NoInput {
			return errs.InputRequired("text or media")
		}
		return errs.New(errs.ExitUsage, "USAGE", "a post needs --text and/or --image/--video",
			`knit post create --text "hello" --allow-mutations`)
	}

	plan := c.plan()
	hash := planHash(plan)
	if rt.Cfg.DryRun {
		out := map[string]any{"dryRun": true, "plan": plan, "hash": hash}
		if q, err := rt.API.PublishingLimit(rt.Ctx); err == nil {
			out["quotaRemaining"] = q.Remaining
		}
		return rt.Out.Emit(out)
	}
	if c.Apply != "" && c.Apply != hash {
		return errs.New(errs.ExitUsage, "PLAN_MISMATCH",
			"the --apply hash does not match the current plan",
			"re-run with --dry-run to get the current hash, then --apply <hash>")
	}

	p, err := rt.API.Publish(rt.Ctx, api.PublishReq{
		Text: c.Text, ImageURLs: c.Image, VideoURLs: c.Video, Link: c.Link,
		ReplyToID: c.ReplyTo, QuotePostID: c.Quote, Topic: c.Topic, ReplyControl: c.ReplyAudience,
	})
	if err != nil {
		return err
	}
	return rt.Out.Emit(map[string]any{"ok": true, "id": p.ID, "permalink": deref(p.Permalink)})
}

// --- post repost ------------------------------------------------------------

type PostRepostCmd struct {
	ID string `arg:"" help:"Media id to repost."`
}

func (c *PostRepostCmd) Run(rt *Runtime) error {
	if err := rt.Guard("post repost"); err != nil {
		return err
	}
	if rt.Cfg.DryRun {
		return rt.Out.Emit(map[string]any{"dryRun": true, "action": "post.repost", "id": c.ID})
	}
	id, err := rt.API.Repost(rt.Ctx, c.ID)
	if err != nil {
		return err
	}
	return rt.Out.Emit(map[string]any{"ok": true, "id": id})
}

// --- post delete ------------------------------------------------------------

type PostDeleteCmd struct {
	ID string `arg:"" help:"Media id to delete."`
}

func (c *PostDeleteCmd) Run(rt *Runtime) error {
	if err := rt.Guard("post delete"); err != nil {
		return err
	}
	if rt.Cfg.DryRun {
		return rt.Out.Emit(map[string]any{"dryRun": true, "action": "post.delete", "id": c.ID})
	}
	res, err := rt.API.DeletePost(rt.Ctx, c.ID)
	if err != nil {
		return err
	}
	return rt.Out.Emit(res)
}

// planHash is a stable content hash of a publish plan (reviewed-artifact = approval).
func planHash(plan map[string]any) string {
	b, _ := json.Marshal(plan)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
