package templates

import (
	"github.com/kaiagaoo/PickMem/internal/vault"
)

// StarterTag marks every note seeded by the starter template, so users
// can find them (`pickmem list`, picker filter: "starter") and sweep the
// leftovers once they've filled in what they want.
const StarterTag = "starter"

// StarterNote is one fill-in-the-blank note the starter template seeds.
// Blanks are written as `____` — the user replaces them in Obsidian or
// via `pickmem edit` and deletes the notes they don't need.
type StarterNote struct {
	Group string
	Label string
	Body  string
}

// StarterNotes is the shipped fill-in set: one skeleton note per leaf
// group of the starter taxonomy, so a fresh vault reads as a form to
// complete rather than an empty tree. Bodies are phrased so that a
// filled-in note reads naturally when assembled into model context.
var StarterNotes = []StarterNote{
	{"about/identity", "basics",
		"Name: ____\nBased in: ____\nWorks as: ____ at ____"},
	{"about/preferences", "communication preferences",
		"Prefers replies that are: ____ (concise / detailed / step-by-step)\nTools used daily: ____\nDon't suggest: ____"},
	{"about/health", "health basics",
		"Allergies: ____\nOngoing conditions: ____\nCurrent medications: ____"},
	{"work/role", "current role",
		"Role: ____ at ____ (since ____)\nMain responsibilities: ____"},
	{"work/projects", "current project",
		"Project: ____\nGoal: ____\nNext milestone: ____"},
	{"work/stack", "tech stack",
		"Languages / frameworks: ____\nEditor & environment: ____\nInfra / platforms: ____"},
	{"work/contacts", "key contact",
		"Name: ____\nRole / relationship: ____\nContext: ____"},
	{"finance/income", "income",
		"Monthly income: ____\nOther sources: ____"},
	{"finance/bills", "recurring bills",
		"Housing: ____ / month\nSubscriptions & utilities: ____"},
	{"finance/goals", "financial goal",
		"Saving for: ____\nTarget: ____ by ____"},
	{"home/housing", "housing situation",
		"Type: ____ (apartment / house), ____ (rent / own)\nLocation: ____"},
	{"home/logistics", "logistics",
		"Time zone: ____\nCommute / transport: ____"},
	{"relationships/family", "family",
		"Household / family: ____"},
	{"relationships/friends", "close friends",
		"Close friends and their context: ____"},
	{"relationships/dates", "important dates",
		"____'s birthday: ____\nAnniversary: ____"},
	{"learning/topics", "currently learning",
		"Topic: ____\nCurrent level: ____\nGoal: ____"},
	{"learning/resources", "learning resources",
		"Preferred formats & go-to resources: ____"},
	{"projects", "personal project",
		"Project: ____\nStatus: ____\nNext step: ____"},
}

// SeedNotes creates the starter notes in the vault as active notes with
// fresh ids. A note whose (group, label) already exists is skipped, so
// re-running (e.g. `init --force`) never duplicates a note the user has
// been filling in. Returns how many notes were created.
func SeedNotes(s *vault.Store) (int, error) {
	existing := map[[2]string]bool{}
	for _, n := range s.List() {
		existing[[2]string{n.Group, n.Label}] = true
	}
	created := 0
	for _, sn := range StarterNotes {
		if existing[[2]string{sn.Group, sn.Label}] {
			continue
		}
		_, err := s.Add(&vault.Note{
			Frontmatter: vault.Frontmatter{
				Label: sn.Label,
				Group: sn.Group,
				Tags:  []string{StarterTag},
			},
			Body: sn.Body,
		})
		if err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}
