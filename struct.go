package main

// Node struct
type Node struct {
	Caption           string              `json:"caption"`
	Code              string              `json:"code"`
	CommentsDisabled  bool                `json:"comments_disabled"`
	Date              int                 `json:"date"`
	DisplaySrc        string              `json:"display_src"`
	ID                string              `json:"id"`
	IsVideo           bool                `json:"is_video"`
	ThumbnailSrc      string              `json:"thumbnail_src"`
	ThumbnailResource []ThumbnailResource `json:"thumbnail_resources"`
	Comments          struct {
		Count int `json:"Count"`
	} `json:"comments"`
	Dimensions struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"dimensions"`
	Likes struct {
		Count int `json:"Count"`
	} `json:"likes"`
	Owner struct {
		ID string `json:"id"`
	} `json:"owner"`
}

// ThumbnailResource struct
type ThumbnailResource struct {
	Src          string `json:"src"`
	ConfigWidth  int    `json:"config_width"`
	ConfigHeight int    `json:"config_height"`
}

type media struct {
	Count    int    `json:"count"`
	Nodes    []Node `json:"nodes"`
	PageInfo struct {
		EndCursor       string `json:"end_cursor"`
		HasNextPage     bool   `json:"has_next_page"`
		HasPreviousPage bool   `json:"has_previous_page"`
		StartCursor     string `json:"start_cursor"`
	} `json:"page_info"`
}

type profile struct {
	Biography          string `json:"biography"`
	FullName           string `json:"full_name"`
	HasRequestedViewer bool   `json:"has_requested_viewer"`
	ID                 string `json:"id"`
	IsPrivate          bool   `json:"is_private"`
	Media              media  `json:"media"`
	ProfilePicURL      string `json:"profile_pic_url"`
	ProfilePicURLHd    string `json:"profile_pic_url_hd"`
	Username           string `json:"username"`
	FollowedBy         struct {
		Count int `json:"count"`
	} `json:"followed_by"`
	Follows struct {
		Count int `json:"count"`
	} `json:"follows"`
}

type profilepage struct {
	User profile `json:"user"`
}

// IGData struct
type IGData struct {
	Code      string `json:"country_code"`
	EntryData struct {
		ProfilePage []profilepage `json:"ProfilePage"`
	} `json:"entry_data"`
}

type queryData struct {
	Status string `json:"status"`
	Media  media  `json:"media"`
}
