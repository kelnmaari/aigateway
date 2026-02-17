%define name aigateway
# Version and release passed via --define from CI
# For tags: version=X.Y.Z, release=1
# For branches: version=X.Y.Z, release=branch.jobid

# Define systemd unit directory
%define _unitdir /etc/systemd/system

Name:           %{name}
Version:        %{version}
Release:        %{release}%{?dist}
Summary:        OpenAI-compatible AI Gateway for multi-provider LLM inference

License:        MIT
URL:            https://github.com/kelnmaari/ollama-openai-proxy

# Disable debug package and build id
%global debug_package %{nil}
%global _build_id_links none

# Don't strip binaries
%define __strip /bin/true

# No dependencies - user manages runtime environment

%description
AIGateway provides an OpenAI-compatible API for multi-provider LLM inference
using container-based backends (vLLM, llama.cpp, SGLang, TGI) and external
providers (OpenAI, Anthropic, Gemini).
Features include multi-tenant support, API key management, GitLab MR reviews,
and a modern web UI.

Requires container runtime: docker-ce or podman (install separately).

%prep
# Nothing to prepare - binary is pre-built

%build
# Nothing to build - binary is pre-built

%install
# Create directories
mkdir -p %{buildroot}/opt/aigateway/bin
mkdir -p %{buildroot}/opt/aigateway/configs
mkdir -p %{buildroot}/opt/aigateway/data
mkdir -p %{buildroot}/opt/aigateway/web
mkdir -p %{buildroot}%{_unitdir}

# Install binary
install -m 755 %{_sourcedir}/aigateway-linux-amd64 %{buildroot}/opt/aigateway/bin/aigateway-linux-amd64

# Install systemd service
install -m 644 %{_sourcedir}/oop.service %{buildroot}%{_unitdir}/oop.service

%files
%defattr(-,root,root,-)
/opt/aigateway/bin/aigateway-linux-amd64
%{_unitdir}/oop.service
%dir /opt/aigateway
%dir /opt/aigateway/bin
%dir /opt/aigateway/configs
%dir /opt/aigateway/data
%dir /opt/aigateway/web

%pre
# Pre-install: nothing special needed

%post
# Post-install: reload systemd and restart service
systemctl daemon-reload
systemctl reset-failed oop 2>/dev/null || true
systemctl enable oop 2>/dev/null || true
systemctl restart oop

echo ""
echo "=============================================="
echo " AIGateway installed/updated!"
echo "=============================================="
sleep 2
systemctl status oop --no-pager || true
echo ""
echo " Logs: journalctl -u oop -f"
echo ""

%preun
# Pre-uninstall: stop service if running
if [ $1 -eq 0 ]; then
    systemctl stop oop 2>/dev/null || true
    systemctl disable oop 2>/dev/null || true
fi

%postun
# Post-uninstall: reload systemd
systemctl daemon-reload

%changelog
* %(date "+%a %b %d %Y") AIGateway Team <team@example.com> - %{version}-%{release}
- Automated build from CI/CD
