variable "services" {
	description = "Consul services monitored by Consul NIA (protocol v0)"
	type = map(object({
		# Name of the service
		name = string
		# Description of the service
		description = string
		# List of addresses for instances of the service by IP and port
		addresses = list(object({
			address = string
			port = number
		}))
	}))
}
