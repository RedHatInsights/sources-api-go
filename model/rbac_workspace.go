package model

type Workspace struct {
	ID 	 string `json:"id"`
	Name string `json:"name"`
}

type WorkspaceResponse struct{
	Data []Workspace `json:"data"`
}
