package descriptors

type MenuItem struct {
	Icon   string `json:"icon"`
	Label  string `json:"label"`
	Url    string `json:"url"`
	IsHref bool   `json:"isHref"`
}

type Menu struct {
	Name      string     `json:"name"`
	MenuItems []MenuItem `json:"menuItems"`
}
