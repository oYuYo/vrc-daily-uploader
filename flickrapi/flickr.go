package flickrapi

type Config struct {
	ApiKey         string
	UserId         string
	SearchEndPoint string
	SearchMethod   string
	ImageEndPoint  string
}

type Photo struct {
	Id       string `json:"id"`
	Owner    string `json:"owner"`
	Secret   string `json:"secret"`
	Server   string `json:"server"`
	Farm     int    `json:"farm"`
	Title    string `json:"title"`
	IsPublic int    `json:"ispublic"`
	IsFriend int    `json:"isfriend"`
	IsFamily int    `json:"isfamily"`
}

type Photos struct {
	Page      int     `json:"page"`
	Pages     int     `json:"pages"`
	Perpage   int     `json:"perpage"`
	Total     int     `json:"total"`
	PhotoList []Photo `json:"photo"`
}

type PhotosSearchJson struct {
	Photos Photos `json:"photos"`
	Stat   string `json:"stat"`
}
