package mongo

// Operations - list of allowed operations for transactions
var Operations = []string{
	"updateOne",
	"insertOne",
	"insertMany",
	"updateMany",
}
