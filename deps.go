package main

func buildNotifier(conf *Config) (Notifier, error) {
	mqtt, err := NewMqttClient(conf.Mqtt.Broker, conf.Mqtt.ClientId, conf.Mqtt.Topic, conf.Mqtt.TlsConfig())
	if err != nil {
		return nil, err
	}

	return mqtt, nil
}

func buildProbers(conf *Config) (map[string]Prober, error) {
	ret := map[string]Prober{}

	for _, probe := range conf.Probes {
		p, err := NewPinger(probe)
		if err != nil {
			return nil, err
		}
		ret[probe.Name] = p
	}

	return ret, nil
}
