// The onboarding template library. Each domain is a category the user can
// choose; each template is a suggested memory with a guiding question and an
// optional example the user can one-click prefill. Groups mirror the starter
// taxonomy that `pickmem init` uses (about/*, work/*, finance/*, home/*,
// relationships/*, learning/*), so a web-built vault matches a CLI one.

export interface Template {
  field: string; // stable key for the answer map
  label: string; // becomes the item label
  group: string; // where it's filed
  question: string; // shown to the user
  example?: string; // one-click prefill / placeholder hint
}

export interface Domain {
  key: string;
  chip: string;
  hint?: string;
  templates: Template[];
}

export const DOMAINS: Domain[] = [
  {
    key: "identity",
    chip: "About you",
    hint: "the basics",
    templates: [
      { field: "id_name", label: "Name", group: "about/identity", question: "What should an assistant call you?", example: "Kaia" },
      { field: "id_location", label: "Location", group: "about/identity", question: "Where are you based?", example: "Lisbon, Portugal" },
      { field: "id_occupation", label: "Occupation", group: "about/identity", question: "What do you do?", example: "Product engineer" },
      { field: "id_languages", label: "Languages", group: "about/identity", question: "Languages you speak or want replies in?", example: "English, Portuguese" },
    ],
  },
  {
    key: "prefs",
    chip: "Preferences & style",
    hint: "how I like things done",
    templates: [
      { field: "pref_length", label: "Response length", group: "about/preferences", question: "Concise or detailed by default?", example: "Concise; expand only when I ask" },
      { field: "pref_tone", label: "Tone", group: "about/preferences", question: "Preferred tone?", example: "Direct, no fluff, a little dry humor" },
      { field: "pref_format", label: "Formatting", group: "about/preferences", question: "How should answers be formatted?", example: "Short paragraphs + bullets; code in fenced blocks" },
      { field: "pref_avoid", label: "Things to avoid", group: "about/preferences", question: "Anything an assistant should NOT do?", example: "No emojis, don't hedge, don't restate my question" },
      { field: "pref_level", label: "Expertise level", group: "about/preferences", question: "What level should it assume?", example: "Senior engineer — skip the basics, cite sources" },
      { field: "pref_env", label: "Editor & environment", group: "about/preferences", question: "Tools/environment you work in?", example: "Neovim, macOS, zsh" },
    ],
  },
  {
    key: "work",
    chip: "Work & projects",
    templates: [
      { field: "work_role", label: "Role", group: "work/role", question: "What's your role?", example: "Product engineer at Acme, since 2023" },
      { field: "work_project", label: "Current project", group: "work/projects", question: "What are you working on right now?", example: "Redesigning onboarding; ship by Q3" },
      { field: "work_stack", label: "Tech stack", group: "work/stack", question: "Main languages / frameworks / tools?", example: "TypeScript, React, Go; deploy on Fly.io" },
      { field: "work_goal", label: "Quarterly goal", group: "work/projects", question: "A goal for this quarter?", example: "Grow activation by 15%" },
      { field: "work_style", label: "Working style", group: "work/role", question: "How do you work best?", example: "Deep-focus mornings; async-first; no meetings before 11" },
      { field: "work_contact", label: "Key contact", group: "work/contacts", question: "Someone at work worth remembering?", example: "Priya — my manager, owns the roadmap" },
    ],
  },
  {
    key: "money",
    chip: "Money & logistics",
    templates: [
      { field: "money_income", label: "Income", group: "finance/income", question: "Any income facts to keep handy?", example: "Salary + freelance; invoices monthly" },
      { field: "money_bills", label: "Recurring bills", group: "finance/bills", question: "Regular bills or subscriptions?", example: "Rent $2k; Figma, GitHub, Spotify" },
      { field: "money_goal", label: "Financial goal", group: "finance/goals", question: "A money goal?", example: "Save a 6-month emergency fund by year end" },
      { field: "money_home", label: "Home base", group: "home/logistics", question: "Home city / timezone?", example: "Lisbon; WET (UTC+0)" },
      { field: "money_housing", label: "Housing", group: "home/housing", question: "Your housing situation?", example: "Renting a 2-bed apartment downtown" },
    ],
  },
  {
    key: "health",
    chip: "Health & habits",
    templates: [
      { field: "health_allergies", label: "Allergies", group: "about/health", question: "Any allergies?", example: "Penicillin; peanuts" },
      { field: "health_conditions", label: "Conditions", group: "about/health", question: "Ongoing conditions to keep in mind?", example: "Mild asthma" },
      { field: "health_diet", label: "Diet", group: "about/health", question: "Dietary preferences or restrictions?", example: "Vegetarian; low caffeine after noon" },
      { field: "health_routine", label: "Routine", group: "about/health", question: "A daily habit or routine?", example: "Runs 3x/week; sleeps around 11pm" },
    ],
  },
  {
    key: "people",
    chip: "People & relationships",
    templates: [
      { field: "ppl_family", label: "Household / family", group: "relationships/family", question: "Family or household to know?", example: "Partner Sam; two kids (8, 5)" },
      { field: "ppl_friend", label: "Close friend", group: "relationships/friends", question: "A close friend and their context?", example: "Jordan — college friend, lives in Berlin" },
      { field: "ppl_date", label: "Important date", group: "relationships/dates", question: "A date worth remembering?", example: "Sam's birthday: March 4" },
      { field: "ppl_pet", label: "Pets", group: "relationships/family", question: "Any pets?", example: "Dog named Miso" },
    ],
  },
  {
    key: "learning",
    chip: "Learning & interests",
    templates: [
      { field: "learn_goal", label: "Learning goal", group: "learning/topics", question: "What are you learning right now?", example: "Rust — intermediate; building a CLI" },
      { field: "learn_interests", label: "Interests", group: "learning/topics", question: "Topics or hobbies you care about?", example: "Bouldering, film photography, jazz" },
      { field: "learn_formats", label: "How you learn", group: "learning/resources", question: "Preferred way to learn?", example: "Hands-on projects + reference docs, not video" },
    ],
  },
];
