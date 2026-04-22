package models

// OnboardingOption represents a possible answer
type OnboardingOption struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	RiskScore  int    `json:"-"` // hidden from frontend
	HorizonYrs int    `json:"-"` // hidden from frontend
}

// OnboardingQuestion represents a question and its options
type OnboardingQuestion struct {
	ID      string             `json:"id"`
	Text    string             `json:"text"`
	Options []OnboardingOption `json:"options"`
}

// OnboardingSubmitRequest is the payload received from the frontend
type OnboardingSubmitRequest struct {
	// map "question_id" -> "option_id"
	Answers map[string]string `json:"answers" binding:"required"`
}
