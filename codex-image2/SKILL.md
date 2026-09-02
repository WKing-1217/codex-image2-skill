---
name: codex-image2
description: Generate or edit raster images through a configurable OpenAI-compatible API using gpt-image-2. Use for image generation, image editing, visual variants, product shots, covers, illustrations, backgrounds, or first-time secure API setup for this image CLI.
---

# Codex Image2

Use the bundled native CLI for image generation and editing. Prefer it over the built-in `image_gen` tool whenever this skill is active. Normal use requires no Python, Node.js, Go, or package installation.

## Select the executable

- Windows x64: `bin/codex-image2-windows-amd64.exe`
- Windows ARM64: `bin/codex-image2-windows-arm64.exe`
- macOS Intel: `bin/codex-image2-darwin-amd64`
- macOS Apple Silicon: `bin/codex-image2-darwin-arm64`

Resolve these paths relative to this `SKILL.md`. On macOS, run `chmod +x <executable>` if execute permission was not preserved. Do not compile from source during normal use.

## First-time setup on Windows

When the user asks to **install and initialize**, initialize, or reconfigure this skill, ALWAYS execute `setup` after installation/update. This includes an already installed skill and an existing configuration. Do not skip the window based on `configured: true`, `initialized: true`, environment variables, or a successful dry run. Do not merely print a setup command for the user when you can execute it yourself.

For an ordinary image request on Windows, run `status` first. If `initialized` is not true (including a missing field in an older executable), run `setup`. `configured` only means configuration values exist; it does not prove authentication or image generation works.

```powershell
& "<skill-dir>\bin\codex-image2-windows-amd64.exe" setup
```

The executable opens a local window for the API URL and API Key. Tell the user to enter the Key only in that window. Never ask for or accept the Key in chat, a prompt, a command argument, a file, or a log.

Tell the user that setup will generate one low-quality test image and their provider may charge for it. The setup command saves the URL in the user configuration directory and the Key in Windows Credential Manager as unverified, then makes one real image-generation request. Only after decoding the returned image, saving it, and verifying the saved file does it record `initialized: true`. `setup --no-test` is not supported.

Wait for the command to finish. Only report initialization complete when `setup` exits successfully, returns `initialized: true`, and you have inspected and displayed the newly generated image from `test_image.outputs` using its absolute path. A saved key, `status`, `--dry-run`, an HTTP 200 without a valid image, or an old test image is not sufficient. No Codex restart is required for this configuration or test.

On cancellation, report that this initialization was cancelled; do not call it successful. On a failed test, say initialization is incomplete and explain the sanitized error. During initialization, if the URL/Key needs correction, explain the problem and open `setup` again so the user can correct it locally. Never repeat requests with unchanged invalid credentials or loop through setup automatically; if the same error recurs, stop and ask the user to resolve it before retrying. Do not promise to finish while credentials, permissions, or the provider remain unavailable.

If filesystem or network access is blocked, explain that the user must manually grant the needed permission or switch the task to full access, then continue after they do so. The skill cannot change its own permissions. Do not repeatedly retry while permission remains blocked.

The Windows setup window is not available on macOS in this release. On macOS, require `CODEX_API_URL` and `CODEX_API_KEY` to already be present in the executable's environment; never collect them in chat. Environment-only `status` does not persist an initialization marker. To validate macOS initialization, run a real `generate` request for one low-quality test image, inspect it, and display it; do not claim the Windows setup workflow succeeded.

## Workflow

1. Determine whether the request is a new image, an edit, or multiple distinct assets or variants.
2. Collect the intended use, exact text, visual constraints, and avoid items that materially affect the result.
3. Preserve detailed prompts. Clarify generic prompts without inventing brands, people, slogans, or unrelated objects.
4. Run `generate`, `edit`, or `generate-batch` as appropriate.
5. Inspect each output for subject, composition, text accuracy, constraints, and artifacts.
6. Make a targeted revision when needed and re-check it.
7. Display the final image and report its absolute path, prompt, size, quality, and model.

## Prompt structure

Use only relevant lines:

```text
Asset type: <where the image will be used>
Primary request: <the user's request>
Scene/backdrop: <environment>
Subject: <main subject>
Style/medium: <photo, illustration, 3D, etc.>
Composition/framing: <camera angle, crop, placement, negative space>
Lighting/mood: <lighting and mood>
Color palette: <palette notes>
Text (verbatim): "<exact text>"
Constraints: <must keep or include>
Avoid: <must not include>
```

Do not add detail merely to fill the schema. Quote requested image text verbatim and request exact rendering.

## Generate one image

```powershell
& "<skill-dir>\bin\codex-image2-windows-amd64.exe" generate `
  --prompt "A small blue nebula in a glass bottle, studio product photo" `
  --size 1024x1024 `
  --quality auto `
  --out "output/imagegen/nebula.png"
```

Use `--prompt-file` for long prompts. Use `--n` only for variants of the same prompt. Use separate calls or a batch for distinct assets.

## Edit an image

Inspect each input image before editing. State its role and repeat invariants so unrelated details do not drift.

```powershell
& "<skill-dir>\bin\codex-image2-windows-amd64.exe" edit `
  --image "input/product.png" `
  --prompt "Replace only the background with a warm studio backdrop. Keep the product, label, proportions, and edges unchanged." `
  --quality auto `
  --out "output/imagegen/product-edited.png"
```

Repeat `--image` for multiple reference or compositing inputs. Use `--mask mask.png` for a compatible localized PNG mask. Preserve originals and write edits to a new path.

## Generate a batch

Read [references/batch-format.md](references/batch-format.md) before preparing a batch, then run:

```powershell
& "<skill-dir>\bin\codex-image2-windows-amd64.exe" generate-batch `
  --input "tmp/imagegen/jobs.jsonl" `
  --out-dir "output/imagegen" `
  --concurrency 2
```

## Configuration and safety

- Prefer the Windows secure setup over environment variables for ordinary users.
- Saved secure configuration takes priority as one URL/Key pair. Environment variables are a fallback only when no saved URL exists; never combine values from the two sources. If a saved pair is incomplete or a save was interrupted, run setup rather than silently using an environment key.
- `status` never contacts the API. `initialized: true` records a previous successful setup test, not a guarantee that a key remains valid forever. Legacy saved settings and environment-only settings start unverified.
- Never silently choose or display an API host in the setup window. Keep the API URL field empty every time setup opens, even when a saved URL or `CODEX_API_URL` exists, and require the user to enter the address supplied by their API provider.
- Allow HTTP only for localhost; require HTTPS for remote hosts.
- Default to model `gpt-image-2`, size `1024x1024`, and quality `auto`.
- Use `--dry-run` to validate a request without network access or a Key.
- Save project assets inside the current project. The default output directory is `output/imagegen/`.
- Never overwrite files unless the user explicitly authorizes it and `--force` is passed.
- Native transparent output is not guaranteed. Do not promise it or silently switch tools.
- Run `reset --yes` only when the user explicitly asks to remove the saved configuration.

## Failure handling

- Do not retry authentication, validation, permission, or ordinary 4xx errors.
- For 401 during initialization, follow the correction flow above and execute the local setup window instead of only handing the user a command. For 401 during ordinary generation, explain the authentication failure and ask whether to reopen setup. Never ask for the Key in chat.
- For 403 or 404, explain that the subscription group, model access, or API URL may not support `gpt-image-2`.
- Normal image commands retry timeouts, 429, 500, 502, 503, 504, and 524 with bounded backoff. Setup makes only one generation attempt per submission to reduce duplicate charges; after a timeout, say the server may still have processed the request before offering a retry.
- On repeated timeout, suggest `--quality low`, a square size, fewer concurrent jobs, or a later retry.
- Never expose an Authorization header, full Key, server response body, or secrets from the environment.
