# Your PickMem vault

This folder is your memory vault. Each `.md` file with frontmatter is one
**memory item**; the folders group them. PickMem shows you these groups when
you run `pickmem pick` and lets you choose which items reach the model for a
given session.

> PickMem ignores this README (and any note without a `---` frontmatter block),
> so it's safe to keep here as a guide — it will never be sent to a model.

## The starter taxonomy

You don't have to use all of these, and you can rename, nest, or delete any of
them — **the `group:` field in each note's frontmatter is what PickMem actually
reads**, not the folder name. The folders are just a starting shape.

| Group | For |
|-------|-----|
| `about/identity` | Who you are: where you're from, languages, background. |
| `about/preferences` | How you like to work and communicate; tastes and dislikes. |
| `about/health` | Conditions, medications, diet, fitness, providers. |
| `work/role` | Your job title, responsibilities, company context. |
| `work/projects` | What you're actively working on. |
| `work/stack` | Tools, languages, and infrastructure you use. |
| `work/contacts` | Colleagues, clients, collaborators. |
| `finance/income` | Salary, freelance, invoices, bonuses. |
| `finance/bills` | Recurring expenses, subscriptions, insurance. |
| `finance/goals` | Savings, budget, retirement, investing. |
| `home/housing` | Rent/mortgage, landlord, lease, your place. |
| `home/logistics` | Appliances, maintenance, vehicle. |
| `relationships/family` | Close family. |
| `relationships/friends` | Friends and their context. |
| `relationships/dates` | Birthdays, anniversaries, gift ideas. |
| `learning/topics` | Subjects you're studying. |
| `learning/resources` | Books, papers, courses, podcasts. |
| `projects` | Side projects and hobbies. |

Groups nest with `/`. You can go deeper any time — e.g. `work/projects/acme` —
and the picker will show it indented under its parent.

## Fill in the blanks

Each group starts with one **fill-in-the-blank note** (tagged `starter`), so
this vault begins as a form, not an empty tree. Open any of them — in Obsidian
or with `pickmem edit <id>` — and replace the `____` blanks:

```
Monthly income: ____        →   Monthly income: $8k base + quarterly bonus
Other sources: ____         →   Other sources: freelance, ~$1k/mo
```

Fill in only the ones you care about; delete the rest with
`pickmem rm <id> --yes`. A skipped blank is harmless — you simply won't pick
that note. (If you delete one and want it back, `pickmem init <path> --force`
restores missing skeletons without touching notes you've filled in.)

## Adding more memories

```bash
# a fact the model should know about you
pickmem add --label "prefers direct feedback" --group about/preferences \
  --body "I want blunt, specific feedback — skip the hedging."
```

Then run `pickmem pick`, select what's relevant to your task, and press `enter`.

## Reorganizing later

To move a note to a different group, change its `group:` field in the note's
frontmatter (in Obsidian or your editor) — that field is what PickMem reads, so
the note follows the new group even if the file stays where it is. You can also
rearrange these folders freely in Obsidian.

PickMem only ever creates and moves files it manages. It never edits notes
you've written yourself — edit those freely in Obsidian.
