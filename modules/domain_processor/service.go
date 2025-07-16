package domain_processor

type Service struct {}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ProcessDomain(msg DomainMessage) string {
	return "processed_by_domain_service_" + msg.Value
}