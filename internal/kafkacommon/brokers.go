package kafkacommon

import "strings"

func NormalizeBrokers(brokers []string) []string {
	normalized := make([]string, 0, len(brokers))

	for _, broker := range brokers {
		broker = strings.TrimSpace(broker)
		if broker != "" {
			normalized = append(normalized, broker)
		}
	}

	return normalized
}
