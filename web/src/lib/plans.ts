export type PlanTask = { title: string; description: string };
export type Starter = {
  id: string;
  name: string;
  goal: string;
  policy: "manual" | "verified";
  tasks: string[];
};
export const starters: Starter[] = [
  {
    id: "app",
    name: "Build my first app",
    goal: "Build a simple app that solves one problem in my daily life. Help me decide what to build and try a small working version.",
    policy: "manual",
    tasks: [
      "Describe the problem and who needs help",
      "Choose the smallest useful version",
      "Ask AI to build a working first version",
      "Try it with realistic examples and fix what fails",
      "Publish the app, save its link, and test it on another device",
    ],
  },
  {
    id: "shifts",
    name: "Manage shifts & tips",
    goal: "Create a simple routine for tracking my shifts, tips, and upcoming commitments.",
    policy: "manual",
    tasks: [
      "Write down this week’s shifts and commitments",
      "Choose what to record after each shift",
      "Ask AI for a simple weekly tracker",
      "Try the tracker for one shift and adjust it",
    ],
  },
  {
    id: "household",
    name: "Organize my household",
    goal: "Make the coming week easier to manage with one place for meals, appointments, and household jobs.",
    policy: "manual",
    tasks: [
      "Collect this week’s appointments and deadlines",
      "Choose a realistic meal and shopping plan",
      "Break household jobs into small next steps",
      "Review the plan with everyone involved",
    ],
  },
  {
    id: "career",
    name: "Take my next career step",
    goal: "Make steady progress toward a better role with a clear plan I can fit around my current commitments.",
    policy: "manual",
    tasks: [
      "Choose one career goal for this month",
      "List experience and strengths to highlight",
      "Ask AI to help tailor my resume to one role",
      "Review the draft and take the next application step",
    ],
  },
  {
    id: "business",
    name: "Grow my business",
    goal: "Turn one business idea into a small, useful offer I can test with a real customer.",
    policy: "manual",
    tasks: [
      "Describe the customer and the problem",
      "Draft one simple offer with AI",
      "Review costs, time, and a realistic price",
      "Ask a potential customer for feedback",
    ],
  },
  {
    id: "product",
    name: "Plan product work",
    goal: "Turn a customer problem into a testable product plan with clear outcomes.",
    policy: "manual",
    tasks: [
      "Write the customer problem and evidence",
      "Define the smallest experiment",
      "Agree on how we will judge the outcome",
      "Review results and decide what to do next",
    ],
  },
  {
    id: "engineering",
    name: "Ship software or infrastructure",
    goal: "Deliver a small, safe change with visible agent progress and verified release evidence.",
    policy: "verified",
    tasks: [
      "Define the change and acceptance criteria",
      "Implement and report progress",
      "Test the change and recovery plan",
      "Review and merge or deploy the change",
    ],
  },
];
export function parseTaskLines(text: string): PlanTask[] {
  return text
    .split("\n")
    .map((s) => s.replace(/^\s*(?:[-*•]|\d+[.)])\s*/, "").trim())
    .filter(Boolean)
    .map((title) => ({ title, description: "" }));
}
export function planPrompt(goal: string) {
  return `Help me make progress on this goal: ${goal}\n\nAsk up to three questions if needed. Use plain language and suggest a small first step. Then give me no more than 8 concrete tasks, one task per line, without headings. Do not claim to have performed actions or connected to my board. Avoid asking me to share passwords, keys, or private information. If I am building an app, help me choose a usable first version and explain what I need to build, test, and publish it.`;
}
