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

Before the first image request, run `status`. If `configured` is false, run `setup`:

```powershell
& "<skill-dir>\bin\codex-image2-windows-amd64.exe" setup
```

The executable opens a local window for the API URL and API Key. Tell the user to enter the Key only in that window. Never ask for or accept the Key in chat, a prompt, a command argument, a file, or a log.

The setup command saves the URL in the user configuration directory, saves the Key in Windows Credential Manager, and generates a low-quality test image. Inspect and display the test image from `test_image.outputs`. No Codex restart is required.

If filesystem or network access is blocked, explain that the user must manually grant the needed permission or switch the task to full access, then continue after they do so. The skill cannot change its own permissions. Do not repeatedly retry while permission remains blocked.

The Windows setup window is not available on macOS in this release. On macOS, require `CODEX_API_URL` and `CODEX_API_KEY` to already be present in the executable's environment; never collect them in chat.

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
- Environment variables remain supported and override saved configuration.
- Never silently choose an API host. On first setup, leave the API URL empty and require the user to enter the address supplied by their API provider. A previously saved URL or an explicit `CODEX_API_URL` may be shown again during reconfiguration.
- Allow HTTP only for localhost; require HTTPS for remote hosts.
- Default to model `gpt-image-2`, size `1024x1024`, and quality `auto`.
- Use `--dry-run` to validate a request without network access or a Key.
- Save project assets inside the current project. The default output directory is `output/imagegen/`.
- Never overwrite files unless the user explicitly authorizes it and `--force` is passed.
- Native transparent output is not guaranteed. Do not promise it or silently switch tools.
- Run `reset --yes` only when the user explicitly asks to remove the saved configuration.

## Failure handling

- Do not retry authentication, validation, permission, or ordinary 4xx errors.
- For 401, ask the user to rerun the local setup window and verify the Key.
- For 403 or 404, explain that the subscription group, model access, or API URL may not support `gpt-image-2`.
- The CLI retries timeouts, 429, 500, 502, 503, 504, and 524 with bounded backoff.
- On repeated timeout, suggest `--quality low`, a square size, fewer concurrent jobs, or a later retry.
- Never expose an Authorization header, full Key, server response body, or secrets from the environment.
