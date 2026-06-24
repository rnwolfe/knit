package api

// This file defines the clean, spec-shaped output types (camelCase JSON, append-only) and
// maps the Graph API's snake_case responses onto them. See spec.md §Output schema.

// Field selectors sent to the Graph API.
const (
	postFields    = "id,username,text,media_type,media_url,permalink,timestamp,is_quote_post,link_attachment_url,quoted_post{id}"
	replyFields   = "id,username,text,media_type,media_url,permalink,timestamp,is_quote_post,hide_status,has_replies,reply_audience,replied_to{id},root_post{id}"
	profileFields = "id,username,name,threads_biography,threads_profile_picture_url,is_verified"
)

// Post is the spec-shaped post object.
type Post struct {
	ID                string  `json:"id"`
	Username          string  `json:"username,omitempty"`
	Text              *string `json:"text"`
	MediaType         string  `json:"mediaType,omitempty"`
	MediaURL          *string `json:"mediaUrl"`
	Permalink         *string `json:"permalink"`
	Timestamp         string  `json:"timestamp,omitempty"`
	IsQuotePost       bool    `json:"isQuotePost"`
	ReplyAudience     *string `json:"replyAudience,omitempty"`
	LinkAttachmentURL *string `json:"linkAttachmentUrl"`
	QuotedPostID      *string `json:"quotedPostId"`
}

// Reply extends Post with reply-management fields.
type Reply struct {
	Post
	HideStatus  string  `json:"hideStatus,omitempty"`
	RepliedToID *string `json:"repliedToId"`
	RootPostID  *string `json:"rootPostId"`
	HasReplies  bool    `json:"hasReplies"`
}

// Profile is the spec-shaped profile object.
type Profile struct {
	ID                string `json:"id"`
	Username          string `json:"username,omitempty"`
	Name              string `json:"name,omitempty"`
	Biography         string `json:"biography,omitempty"`
	ProfilePictureURL string `json:"profilePictureUrl,omitempty"`
	IsVerified        bool   `json:"isVerified"`
}

// DeleteResult is the spec-shaped delete response.
type DeleteResult struct {
	OK        bool   `json:"ok"`
	ID        string `json:"id"`
	DeletedID string `json:"deletedId,omitempty"`
	Existed   bool   `json:"existed"`
}

// Quota is the remaining-publish-quota view (from threads_publishing_limit).
type Quota struct {
	Used      int `json:"used"`
	Total     int `json:"total"`
	Remaining int `json:"remaining"`
}

// --- raw Graph shapes + mapping ---------------------------------------------

type idRef struct {
	ID string `json:"id"`
}

type rawMedia struct {
	ID                string  `json:"id"`
	Username          string  `json:"username"`
	Text              *string `json:"text"`
	MediaType         string  `json:"media_type"`
	MediaURL          *string `json:"media_url"`
	Permalink         *string `json:"permalink"`
	Timestamp         string  `json:"timestamp"`
	IsQuotePost       bool    `json:"is_quote_post"`
	ReplyAudience     *string `json:"reply_audience"`
	LinkAttachmentURL *string `json:"link_attachment_url"`
	QuotedPost        *idRef  `json:"quoted_post"`
	HideStatus        string  `json:"hide_status"`
	HasReplies        bool    `json:"has_replies"`
	RepliedTo         *idRef  `json:"replied_to"`
	RootPost          *idRef  `json:"root_post"`
}

func (r *rawMedia) toPost() Post {
	p := Post{
		ID: r.ID, Username: r.Username, Text: r.Text, MediaType: r.MediaType,
		MediaURL: r.MediaURL, Permalink: r.Permalink, Timestamp: r.Timestamp,
		IsQuotePost: r.IsQuotePost, ReplyAudience: r.ReplyAudience,
		LinkAttachmentURL: r.LinkAttachmentURL,
	}
	if r.QuotedPost != nil {
		p.QuotedPostID = &r.QuotedPost.ID
	}
	return p
}

func (r *rawMedia) toReply() Reply {
	rp := Reply{Post: r.toPost(), HideStatus: r.HideStatus, HasReplies: r.HasReplies}
	if r.RepliedTo != nil {
		rp.RepliedToID = &r.RepliedTo.ID
	}
	if r.RootPost != nil {
		rp.RootPostID = &r.RootPost.ID
	}
	return rp
}

type rawProfile struct {
	ID                       string `json:"id"`
	Username                 string `json:"username"`
	Name                     string `json:"name"`
	ThreadsBiography         string `json:"threads_biography"`
	ThreadsProfilePictureURL string `json:"threads_profile_picture_url"`
	IsVerified               bool   `json:"is_verified"`
}

func (r *rawProfile) toProfile() Profile {
	return Profile{
		ID: r.ID, Username: r.Username, Name: r.Name, Biography: r.ThreadsBiography,
		ProfilePictureURL: r.ThreadsProfilePictureURL, IsVerified: r.IsVerified,
	}
}

// rawPaging is the Graph cursor envelope.
type rawPaging struct {
	Paging struct {
		Cursors struct {
			After string `json:"after"`
		} `json:"cursors"`
		Next string `json:"next"`
	} `json:"paging"`
}

// nextCursor returns the forward cursor, or "" at end-of-results.
func (p *rawPaging) nextCursor() string {
	if p.Paging.Next == "" {
		return ""
	}
	return p.Paging.Cursors.After
}

// PageOpts are common list/pagination options.
type PageOpts struct {
	Limit  int
	Cursor string
	Since  string
	Until  string
}

// PublishReq is a normalized publish request (container step 1).
type PublishReq struct {
	Text         string
	ImageURLs    []string
	VideoURLs    []string
	Link         string
	ReplyToID    string
	QuotePostID  string
	Topic        string
	ReplyControl string
}

// SearchOpts parameterizes keyword/topic search.
type SearchOpts struct {
	Query     string
	MediaType string // text|image|video (mapped to API enums)
	Tag       bool   // search_mode=TAG
	PageOpts
}

// AccountInsightsOpts parameterizes account-level insights.
type AccountInsightsOpts struct {
	Metrics []string
	Since   string
	Until   string
}
