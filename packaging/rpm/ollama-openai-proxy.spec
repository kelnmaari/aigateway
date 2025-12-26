%define name ollama-openai-proxy
# Version and release passed via --define from CI
# For tags: version=X.Y.Z, release=1
# For branches: version=X.Y.Z, release=branch.jobid

Name:           %{name}
Version:        %{version}
Release:        %{release}%{?dist}
Summary:        OpenAI-compatible API proxy for local LLM inference

License:        MIT
URL:            https://github.com/kelnmaari/ollama-openai-proxy

# Disable debug package and build id
%global debug_package %{nil}
%global _build_id_links none

# Don't strip binaries
%define __strip /bin/true

# Requirements
Requires:       docker
Requires:       systemd

%description
AIGateway (Ollama OpenAI Proxy) provides an OpenAI-compatible API for local
LLM inference using Docker-based backends (vLLM, llama.cpp, SGLang, TGI).
Features include multi-tenant support, API key management, GitLab MR reviews,
and a modern web UI.

%prep
# Nothing to prepare - binary is pre-built

%build
# Nothing to build - binary is pre-built

%install
# Create directories
mkdir -p %{buildroot}/opt/ollama-openai-proxy/bin
mkdir -p %{buildroot}/opt/ollama-openai-proxy/configs
mkdir -p %{buildroot}/opt/ollama-openai-proxy/data
mkdir -p %{buildroot}/opt/ollama-openai-proxy/web
mkdir -p %{buildroot}%{_unitdir}

# Install binary
install -m 755 %{_sourcedir}/server %{buildroot}/opt/ollama-openai-proxy/bin/server

# Install systemd service
install -m 644 %{_sourcedir}/oop.service %{buildroot}%{_unitdir}/oop.service

%files
%defattr(-,root,root,-)
/opt/ollama-openai-proxy/bin/server
%{_unitdir}/oop.service
%dir /opt/ollama-openai-proxy
%dir /opt/ollama-openai-proxy/bin
%dir /opt/ollama-openai-proxy/configs
%dir /opt/ollama-openai-proxy/data
%dir /opt/ollama-openai-proxy/web

%pre
# Pre-install: nothing special needed

%post
# Post-install: reload systemd and enable service
systemctl daemon-reload
echo ""
echo "=============================================="
echo " Ollama OpenAI Proxy installed successfully!"
echo "=============================================="
echo ""
echo " 1. Create config file:"
echo "    cp /opt/ollama-openai-proxy/configs/production.yaml.example \\"
echo "       /opt/ollama-openai-proxy/configs/dev.yaml"
echo ""
echo " 2. Edit configuration:"
echo "    vim /opt/ollama-openai-proxy/configs/dev.yaml"
echo ""
echo " 3. Start the service:"
echo "    systemctl enable --now oop"
echo ""
echo " 4. Check status:"
echo "    systemctl status oop"
echo "    journalctl -u oop -f"
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

