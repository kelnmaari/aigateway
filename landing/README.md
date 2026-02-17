# AIGateway Landing Page

Static landing page for AIGateway. Fully self-contained — no dependencies, no build step.

## Local Preview

Open `index.html` directly in a browser:

```bash
# macOS
open index.html

# Linux
xdg-open index.html

# Windows
start index.html
```

## Deployment

This is a static HTML file. Deploy it to any static hosting:

- **Nginx** — copy `index.html` to your web root
- **GitHub Pages** — push this directory to a `gh-pages` branch
- **Netlify / Vercel** — point to the `landing/` directory
- **S3 + CloudFront** — upload `index.html` to an S3 bucket

## Customization

### Links

Update the CTA button URLs in `index.html`:

- `#sign-in` — replace with your AIGateway login URL (e.g., `https://ai.example.com/login`)
- `#get-started` — replace with your registration or docs URL
- GitHub link in the CTA footer — update to your repository URL

### Branding

All styles are inline in `<style>` tags. Key CSS variables are at the top of the file in `:root` — colors, border radius, fonts.
