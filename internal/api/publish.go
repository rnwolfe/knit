package api

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/rnwolfe/knit/internal/errs"
)

// Publish runs the two-step container+publish flow (with VIDEO status polling) and returns
// the published post. Single-media/text only for carousels-of-one; multi-image carousels are
// left to a follow-up (the container path supports children but the CLI surface posts one item).
func (c *Client) Publish(ctx context.Context, r PublishReq) (*Post, error) {
	mediaType, mediaURL := "TEXT", ""
	switch {
	case len(r.VideoURLs) > 0:
		mediaType, mediaURL = "VIDEO", r.VideoURLs[0]
	case len(r.ImageURLs) > 0:
		mediaType, mediaURL = "IMAGE", r.ImageURLs[0]
	}

	q := url.Values{"media_type": {mediaType}}
	if r.Text != "" {
		q.Set("text", r.Text)
	}
	switch mediaType {
	case "IMAGE":
		q.Set("image_url", mediaURL)
	case "VIDEO":
		q.Set("video_url", mediaURL)
	}
	if r.Link != "" {
		q.Set("link_attachment", r.Link)
	}
	if r.ReplyToID != "" {
		q.Set("reply_to_id", r.ReplyToID)
	}
	if r.QuotePostID != "" {
		q.Set("quote_post_id", r.QuotePostID)
	}
	if r.Topic != "" {
		q.Set("topic_tag", r.Topic)
	}
	if r.ReplyControl != "" {
		q.Set("reply_control", r.ReplyControl)
	}

	var created idRef
	if err := c.postForm(ctx, c.userID+"/threads", q, &created); err != nil {
		return nil, err
	}
	if mediaType == "VIDEO" {
		if err := c.waitForContainer(ctx, created.ID); err != nil {
			return nil, err
		}
	}

	var published idRef
	pq := url.Values{"creation_id": {created.ID}}
	if err := c.postForm(ctx, c.userID+"/threads_publish", pq, &published); err != nil {
		return nil, err
	}
	// Fetch the published post so callers get the real permalink/fields.
	if p, err := c.GetPost(ctx, published.ID); err == nil {
		return p, nil
	}
	return &Post{ID: published.ID}, nil
}

// waitForContainer polls a media container until it leaves IN_PROGRESS (videos need encoding).
func (c *Client) waitForContainer(ctx context.Context, id string) error {
	const maxWait = 90 * time.Second
	deadline := time.Now().Add(maxWait)
	for {
		var st struct {
			Status       string `json:"status"`
			ErrorMessage string `json:"error_message"`
		}
		if err := c.get(ctx, id, url.Values{"fields": {"status,error_message"}}, &st); err != nil {
			return err
		}
		switch strings.ToUpper(st.Status) {
		case "FINISHED", "PUBLISHED":
			return nil
		case "ERROR", "EXPIRED":
			return errs.New(errs.ExitGeneric, "PUBLISH_FAILED", "media container "+st.Status+": "+st.ErrorMessage,
				"check the media URL is public and a supported format")
		}
		if time.Now().After(deadline) {
			return errs.New(errs.ExitRetry, "PUBLISH_TIMEOUT", "media container not ready after 90s",
				"retry; large videos can take longer to process")
		}
		select {
		case <-ctx.Done():
			return errs.New(errs.ExitCancelled, "CANCELLED", "cancelled while waiting for media", "")
		case <-time.After(5 * time.Second):
		}
	}
}

// Repost reposts a post. Returns the repost's id.
func (c *Client) Repost(ctx context.Context, id string) (string, error) {
	var out idRef
	if err := c.postForm(ctx, id+"/repost", url.Values{}, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

// DeletePost deletes a post. It is idempotent: a not-found target is a soft success
// (existed=false), so an agent's retries don't hard-fail (contract §9).
func (c *Client) DeletePost(ctx context.Context, id string) (*DeleteResult, error) {
	var out struct {
		Success   bool   `json:"success"`
		DeletedID string `json:"deleted_id"`
	}
	err := c.del(ctx, id, url.Values{}, &out)
	if err != nil {
		var ce *errs.CLIError
		if errors.As(err, &ce) && ce.Code == "NOT_FOUND" {
			return &DeleteResult{OK: true, ID: id, Existed: false}, nil
		}
		return nil, err
	}
	return &DeleteResult{OK: true, ID: id, DeletedID: out.DeletedID, Existed: true}, nil
}

// ManageReply hides or unhides a reply on the user's post. Idempotent.
func (c *Client) ManageReply(ctx context.Context, replyID string, hide bool) (*Reply, error) {
	q := url.Values{"hide": {boolString(hide)}}
	if err := c.postForm(ctx, replyID+"/manage_reply", q, nil); err != nil {
		return nil, err
	}
	status := "NOT_HUSHED"
	if hide {
		status = "HIDDEN"
	}
	return &Reply{Post: Post{ID: replyID}, HideStatus: status}, nil
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
