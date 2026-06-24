package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/rnwolfe/knit/internal/errs"
)

// PostCmd groups the post noun-verb subcommands (the primary noun).
// Reads are backed by the placeholder store; cli-implement repoints them at the API client.
type PostCmd struct {
	List   PostListCmd   `cmd:"" help:"List the authenticated user's posts."`
	Get    PostGetCmd    `cmd:"" help:"Get one post by media id."`
	Create PostCreateCmd `cmd:"" help:"Publish a post (mutation; reviewed-artifact via --dry-run/--apply)."`
	Repost PostRepostCmd `cmd:"" help:"Repost a post by media id (mutation)."`
}

// --- post list --------------------------------------------------------------

type PostListCmd struct {
	Since  string `help:"Only posts on/after this ISO-8601 date."`
	Until  string `help:"Only posts on/before this ISO-8601 date."`
	Cursor string `help:"Opaque pagination cursor from a previous nextCursor."`
}

func (c *PostListCmd) Run(rt *Runtime) error {
	posts, err := rt.Store.ListPosts()
	if err != nil {
		return errs.New(errs.ExitConfig, "STORE_ERROR", err.Error(), "check the store path / KNIT_STORE")
	}
	// cli-implement: pass back the API's paging.cursors.after as the nextCursor.
	return rt.Out.EmitEnvelope(posts, "")
}

// --- post get ---------------------------------------------------------------

type PostGetCmd struct {
	ID string `arg:"" help:"Media id of the post."`
}

func (c *PostGetCmd) Run(rt *Runtime) error {
	p, ok, err := rt.Store.GetPost(c.ID)
	if err != nil {
		return errs.New(errs.ExitConfig, "STORE_ERROR", err.Error(), "check the store path")
	}
	if !ok {
		return errs.NotFound("post", c.ID)
	}
	return rt.Out.EmitEnvelope(p, "")
}

// --- post create ------------------------------------------------------------

// PostCreateCmd publishes a post. High-stakes + irreversible + public, so it uses the
// reviewed-artifact = approval pattern (contract §2): `--dry-run` emits the exact plan plus
// a content hash; `--apply <hash>` publishes only if the recomputed hash matches.
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
		return rt.Out.Emit(map[string]any{"dryRun": true, "plan": plan, "hash": hash})
	}
	if c.Apply != "" && c.Apply != hash {
		return errs.New(errs.ExitUsage, "PLAN_MISMATCH",
			"the --apply hash does not match the current plan",
			"re-run with --dry-run to get the current hash, then --apply <hash>")
	}

	// PLACEHOLDER publish. cli-implement performs the two-step container+publish flow and
	// returns the real media id + permalink, plus remaining daily quota.
	p, err := rt.Store.CreatePost(c.Text)
	if err != nil {
		return errs.New(errs.ExitConfig, "STORE_ERROR", err.Error(), "check the store path is writable")
	}
	return rt.Out.Emit(map[string]any{"ok": true, "id": p.ID, "permalink": p.Permalink})
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
	// PLACEHOLDER. cli-implement: verify a repost endpoint exists (spec open item #2); if not, drop this command.
	return rt.Out.Emit(map[string]any{"ok": true, "id": c.ID, "note": "placeholder — endpoint unverified"})
}

// planHash is a stable content hash of a publish plan (reviewed-artifact = approval).
func planHash(plan map[string]any) string {
	b, _ := json.Marshal(plan)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}
