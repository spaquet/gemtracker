# Gemtracker Claude Code Skills

This directory contains Claude Code skills to enhance gem dependency analysis in your workflow.

## Available Skills

### gemtracker

Direct gemtracker CLI analysis in Claude Code.

**Best for:**
- Automated CI/CD integration
- JSON report parsing
- Machine-readable analysis
- Pre-commit vulnerability scanning

**Features:**
- Structured JSON output
- Vulnerability severity levels
- Multiple vulnerabilities per gem
- CVSS scores
- Insecure source detection

**Usage:**
```
/gemtracker
/gemtracker /path/to/project
/gemtracker . --report json
```

**Installation:**
See [Gemtracker Installation Guide](./gemtracker/INSTALLATION.md)

---

## Comparison: gem-check vs gemtracker

| Feature | gem-check | gemtracker |
|---------|-----------|-----------|
| **Interface** | Interactive guidance | Direct CLI output |
| **Output Format** | Conversational | JSON/Text/CSV |
| **Best Use** | Interactive analysis | Automation/CI |
| **Setup Complexity** | Simple | Requires gemtracker CLI |
| **AI Reasoning** | Built-in recommendations | Raw data for processing |

**Choose gem-check if:** You want guided, interactive analysis with AI recommendations

**Choose gemtracker if:** You need structured data for automation or CI/CD pipelines

---

## Installation

Both skills are included in the gemtracker repository. Install according to the guide for each skill:

- **gem-check**: [.claude/skills/gem-check/SKILL.md](./../.claude/skills/gem-check/SKILL.md)
- **gemtracker**: [skills/gemtracker/INSTALLATION.md](./gemtracker/INSTALLATION.md)

## Support

For issues or questions:

1. **gemtracker CLI issues**: https://github.com/spaquet/gemtracker/issues
2. **Skill usage questions**: Check the skill's INSTALLATION.md
3. **Bug reports**: Include gemtracker version (`gemtracker --version`)

## Contributing

Have improvements? Issues? Pull requests welcome at:
https://github.com/spaquet/gemtracker

## License

All skills inherit the gemtracker license (MIT).
