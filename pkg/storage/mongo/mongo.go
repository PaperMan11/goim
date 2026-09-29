package mongo

type MongoConf struct {
	Uri         string
	Database    string
	EnableCache bool `json:",default=true"`
}
