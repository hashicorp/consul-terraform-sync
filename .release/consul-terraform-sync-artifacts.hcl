# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

schema = 1
artifacts {
  zip = [
    "consul-terraform-sync_${version}_darwin_amd64.zip",
    "consul-terraform-sync_${version}_freebsd_386.zip",
    "consul-terraform-sync_${version}_freebsd_amd64.zip",
    "consul-terraform-sync_${version}_linux_386.zip",
    "consul-terraform-sync_${version}_linux_amd64.zip",
    "consul-terraform-sync_${version}_linux_arm.zip",
    "consul-terraform-sync_${version}_linux_arm64.zip",
    "consul-terraform-sync_${version}_solaris_amd64.zip",
    "consul-terraform-sync_${version}_windows_386.zip",
    "consul-terraform-sync_${version}_windows_amd64.zip",
  ]
  rpm = [
    "consul-terraform-sync-${version_linux}-1.aarch64.rpm",
    "consul-terraform-sync-${version_linux}-1.armv7hl.rpm",
    "consul-terraform-sync-${version_linux}-1.i386.rpm",
    "consul-terraform-sync-${version_linux}-1.x86_64.rpm",
  ]
  deb = [
    "consul-terraform-sync_${version_linux}-1_amd64.deb",
    "consul-terraform-sync_${version_linux}-1_arm64.deb",
    "consul-terraform-sync_${version_linux}-1_armhf.deb",
    "consul-terraform-sync_${version_linux}-1_i386.deb",
  ]
  container = [
    "consul-terraform-sync_default_linux_386_${version}_${commit_sha}.docker.tar",
    "consul-terraform-sync_default_linux_amd64_${version}_${commit_sha}.docker.tar",
    "consul-terraform-sync_default_linux_arm64_${version}_${commit_sha}.docker.tar",
    "consul-terraform-sync_default_linux_arm_${version}_${commit_sha}.docker.tar",
  ]
}
