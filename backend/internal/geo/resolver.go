package geo

type Region struct {
	Country  string
	Province string
	City     string
	ISO      string
}

type Resolver interface {
	Resolve(ip string) (Region, error)
	Close() error
}
