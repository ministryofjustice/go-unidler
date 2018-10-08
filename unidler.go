package main

type Unidle interface {
	Unidle(host string)
}

type Unidler struct{}

func (u *Unidler) Unidle(host string) {
}
