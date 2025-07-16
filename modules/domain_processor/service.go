package domain_processor

import "fmt"

type DomainMessage struct {
	Value     string `json:"value"`
	CompanyID string `json:"companyId"`
}

type Service struct {
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ProcessDomain(msg DomainMessage) string {
	return fmt.Sprintf("www.%s", msg.Value)
}
