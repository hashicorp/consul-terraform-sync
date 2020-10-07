variable "enabled" {
  default = null
  type    = bool
}

variable "format" {
  default = null
  type    = string
}

variable "tags" {
  default = null
  type    = list(any)
}

variable "count" {
  default = null
  type    = number
}
