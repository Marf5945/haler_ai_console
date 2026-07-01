# WD14 Environment Security Notes

This tool uses a small pinned Python dependency set for WD14 tagging:

- `pillow==12.2.0`
- `numpy==2.5.0`
- `onnxruntime==1.27.0`
- `huggingface-hub==1.21.0`
- `pip-audit==2.10.1`

The install script uses `--only-binary=:all:` so `pip` installs published wheels
instead of running arbitrary source builds during setup.

The audit command is:

```bash
image_caption_tagger/.venv/bin/python -m pip_audit
```

Security limits:

- `pip-audit` checks known vulnerability databases for installed Python
  packages. It cannot prove that a package, wheel, or model file is harmless.
- The default WD14 model is downloaded from Hugging Face:
  `SmilingWolf/wd-v1-4-swinv2-tagger-v2`.
- Treat model files and image datasets as external inputs. Keep them out of
  trusted application bundles unless you have reviewed their source and license.
