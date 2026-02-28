package ping

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Pong() string {
	return "pong"
}
