# Vulnerability & Dependency Update Roadmap

**Created**: 2025-01-01
**Last Updated**: 2025-01-01

## Summary

- **NPM Vulnerabilities**: 2 (vite, @sveltejs/kit)
- **NPM Outdated**: 20 packages
- **Go Outdated**: 40+ packages (mostly indirect)

---

## Phase 1: Safe Updates (Minor/Patch) ✅ COMPLETED 2025-01-01

### NPM (web-svelte)

```bash
cd web-svelte
npm update @sveltejs/kit @sveltejs/adapter-static autoprefixer postcss prettier prettier-plugin-svelte svelte-check svelte-sonner @inlang/paraglide-js lucide-svelte typescript svelte
```

| Package | Current | Target | Status |
|---------|---------|--------|--------|
| @sveltejs/kit | 2.16.0 | 2.49.2 | ✅ Done |
| @sveltejs/adapter-static | 3.0.8 | 3.0.10 | ✅ Done |
| autoprefixer | 10.4.20 | 10.4.23 | ✅ Done |
| postcss | 8.4.49 | 8.5.6 | ✅ Done |
| prettier | 3.4.2 | 3.7.4 | ✅ Done |
| prettier-plugin-svelte | 3.3.2 | 3.4.1 | ✅ Done |
| svelte-check | 4.1.1 | 4.3.5 | ✅ Done |
| svelte-sonner | 1.0.6 | 1.0.7 | ✅ Done |
| @inlang/paraglide-js | 2.5.0 | 2.7.1 | ✅ Done |
| lucide-svelte | 0.469.0 | 0.562.0 | ✅ Done |
| typescript | 5.7.2 | 5.9.3 | ✅ Done |
| svelte | 5.16.0 | 5.46.1 | ✅ Done |

### Go Dependencies

```bash
go get -u github.com/gin-gonic/gin@latest github.com/go-playground/validator/v10@latest github.com/fsnotify/fsnotify@latest github.com/coreos/go-oidc/v3@latest
go mod tidy
```

| Package | Current | Target | Status |
|---------|---------|--------|--------|
| gin-gonic/gin | 1.10.1 | 1.11.0 | ✅ Done |
| go-playground/validator | 10.26.0 | 10.30.1 | ✅ Done |
| fsnotify/fsnotify | 1.7.0 | 1.9.0 | ✅ Done |
| coreos/go-oidc/v3 | 3.16.0 | 3.17.0 | ✅ Done |
| bytedance/sonic | 1.13.2 | 1.14.2 | ✅ Done (transitive) |
| golang.org/x/crypto | 0.42.0 | 0.46.0 | ✅ Done (transitive) |
| golang.org/x/net | 0.43.0 | 0.48.0 | ✅ Done (transitive) |

---

## Phase 2: Security Updates ✅ COMPLETED 2025-01-01

### Vite (CVE fixes) ✅

**Updated**: vite 6.0.6 → 6.4.1

**CVEs Fixed**:
- ✅ CVE-2025-32395 (MEDIUM)
- ✅ CVE-2025-31125 (LOW)
- ✅ CVE-2025-46565 (MEDIUM)
- ✅ CVE-2025-62522 (MEDIUM)
- ✅ CVE-2025-58751 (MEDIUM)
- ✅ CVE-2025-58752 (MEDIUM)
- ✅ CVE-2025-24010 (LOW)
- ✅ CVE-2025-30208 (LOW)
- ✅ CVE-2025-31486 (LOW)

### @sveltejs/kit (CVE-2025-32388) ✅

**Updated**: @sveltejs/kit 2.16.0 → 2.49.2

### cookie (GHSA-pxg6-pf52-xh8x) ✅

**Fixed**: Added override in package.json to force cookie ^0.7.2

```json
"overrides": {
  "cookie": "^0.7.2"
}
```

### Final Status: **0 vulnerabilities** 🎉

---

## Phase 3: Major Updates (Breaking Changes) 🔴

**DO NOT update without full testing:**

| Package | Current | Target | Breaking Changes |
|---------|---------|--------|------------------|
| tailwindcss | 3.4.17 | 4.1.18 | Complete rewrite, new config syntax |
| tailwind-variants | 0.3.0 | 3.2.2 | Depends on Tailwind 4 |
| tailwind-merge | 2.6.0 | 3.4.0 | Major API changes |
| bits-ui | 1.0.0-next | 2.14.4 | Beta → Stable 2.x |
| @sveltejs/vite-plugin-svelte | 5.0.3 | 6.2.1 | Requires Vite 7 |
| eslint-plugin-svelte | 2.46.1 | 3.13.1 | ESLint 9 flat config |

### Tailwind 4 Migration Path

1. Read migration guide: https://tailwindcss.com/docs/upgrade-guide
2. Update configuration from `tailwind.config.js` to CSS-based config
3. Update all `@apply` directives
4. Test all components
5. Update tailwind-variants and tailwind-merge

---

## Notes

- **Go dependencies**: Most are indirect, updated transitively
- **NPM**: Vite security fixes are priority
- **Tailwind 4**: Major effort, schedule separately

