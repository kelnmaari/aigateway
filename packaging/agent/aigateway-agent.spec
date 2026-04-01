%define name aigateway-agent
# Version and release passed via --define from CI
# For tags: version=X.Y.Z, release=1
# For branches: version=X.Y.Z, release=branch.jobid

# Define systemd unit directory
%define _unitdir /etc/systemd/system

Name:           %{name}
Version:        %{version}
Release:        %{release}%{?dist}
Summary:        AIGateway Remote Inference Agent for distributed GPU/CPU model serving

License:        MIT
URL:            https://github.com/kelnmaari/ollama-openai-proxy

# Disable debug package and build id
%global debug_package %{nil}
%global _build_id_links none

# Don't strip binaries
%define __strip /bin/true

# No dependencies - user manages runtime environment

%description
AIGateway Agent provides remote inference worker capability for the AIGateway platform.
It manages Docker containers (vLLM, SGLang, TGI, TEI, llama.cpp) on GPU or CPU
servers and exposes an API for the central AIGateway server to control them.

Requires container runtime: docker-ce or podman (install separately).

%prep
# Nothing to prepare - binary is pre-built

%build
# Nothing to build - binary is pre-built

%install
# Create directories
mkdir -p %{buildroot}/opt/aigateway-agent/bin
mkdir -p %{buildroot}/opt/aigateway-agent/configs
mkdir -p %{buildroot}/opt/aigateway-agent/data/models/hf
mkdir -p %{buildroot}/opt/aigateway-agent/data/models/gguf
mkdir -p %{buildroot}/opt/aigateway-agent/logs/containers
mkdir -p %{buildroot}/opt/aigateway-agent/certs
mkdir -p %{buildroot}%{_unitdir}

# Install binary
install -m 755 %{_sourcedir}/aigateway-agent-linux-amd64 %{buildroot}/opt/aigateway-agent/bin/aigateway-agent

# Install systemd service
install -m 644 %{_sourcedir}/aigateway-agent.service %{buildroot}%{_unitdir}/aigateway-agent.service

%files
%defattr(-,root,root,-)
/opt/aigateway-agent/bin/aigateway-agent
%{_unitdir}/aigateway-agent.service
%dir /opt/aigateway-agent
%dir /opt/aigateway-agent/bin
%dir /opt/aigateway-agent/configs
%dir /opt/aigateway-agent/data
%dir /opt/aigateway-agent/data/models
%dir /opt/aigateway-agent/data/models/hf
%dir /opt/aigateway-agent/data/models/gguf
%dir /opt/aigateway-agent/logs
%dir /opt/aigateway-agent/logs/containers
%dir /opt/aigateway-agent/certs

%pre
# Pre-install: nothing special needed

%post
# Post-install: reload systemd
systemctl daemon-reload
systemctl enable aigateway-agent 2>/dev/null || true

# Only restart if config exists (first install: user must create config first)
if [ -f /opt/aigateway-agent/configs/agent.yaml ]; then
    systemctl restart aigateway-agent
    echo ""
    echo "=============================================="
    echo " AIGateway Agent installed/updated!"
    echo "=============================================="
    sleep 2
    systemctl status aigateway-agent --no-pager || true
else
    echo ""
    echo "=============================================="
    echo " AIGateway Agent installed!"
    echo "=============================================="
    echo ""
    echo " IMPORTANT: Create config before starting:"
    echo "   1. Generate config in AIGateway Admin → Workers → Add Worker"
    echo "   2. Copy to: /opt/aigateway-agent/configs/agent.yaml"
    echo "   3. Start:   sudo systemctl start aigateway-agent"
fi
echo ""
echo " Logs: journalctl -u aigateway-agent -f"
echo ""

%preun
# Pre-uninstall: stop service if running
if [ $1 -eq 0 ]; then
    systemctl stop aigateway-agent 2>/dev/null || true
    systemctl disable aigateway-agent 2>/dev/null || true
fi

%postun
# Post-uninstall: reload systemd
systemctl daemon-reload

%changelog
* %(date "+%a %b %d %Y") AIGateway Team <team@example.com> - %{version}-%{release}
- Automated build from CI/CD
