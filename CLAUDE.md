# Engineering Philosophy

This file encodes the design philosophy you must apply to every change you
make in this codebase. It is distilled from Max Kanat-Alexander's *Code
Simplicity*. Internalize it — these are not stylistic preferences, they are
the rules by which work in this repo will be judged.

---

## The single sentence

> **It is more important to reduce the effort of maintenance than it is to
> reduce the effort of implementation, and the effort of maintenance is
> proportional to the complexity of the individual pieces.**

Everything below follows from that.

---

## Six foundational laws

1. **Purpose.** Software exists to help people. Every change should be
   evaluable as "this helps the user do X." If you cannot answer that
   question for a change, the change is probably wrong.

2. **Equation of design.** The desirability of any change is
   `(value_now + future_value) / (effort_to_implement + effort_to_maintain)`.
   Over a long enough timeline, `value_now` and `effort_to_implement` round
   to zero. You are almost always optimizing `future_value / maintenance`.
   Act accordingly.

3. **Change.** The longer code exists, the more probable every piece of it
   will have to change. Design for that.

4. **Defect probability.** The chance of introducing a bug is proportional
   to the size of the change. Small diffs, small blast radius.

5. **Simplicity.** Ease of maintenance is proportional to the simplicity of
   the individual pieces. The whole system can be complex; the pieces must
   not be.

6. **Testing.** You know your software works only to the degree you have
   tested it. Untested code is unknown code.

---

## The Three Flaws — avoid these above all

Most bad code an AI assistant produces falls into one of these three
buckets. Watch yourself for each:

### Flaw 1: Writing code that isn't needed

- Do not write code on speculation. If the user has not asked for it and
  the current task does not require it, do not write it.
- Do not add configuration knobs, plugin points, abstraction layers,
  strategy patterns, or "extension hooks" for needs that have not actually
  arisen. Add them when a second concrete use case shows up, not before.
- If you find dead code (unreachable, unreferenced, commented-out for "in
  case we need it later"), delete it. Version control is the archive.
- Never keep a parameter, flag, field, or branch "for future use." It will
  rot, drift, and eventually mislead someone — possibly you.

### Flaw 2: Not making code easy to change

- Rigid code comes from two sources: (a) too many assumptions about the
  future, and (b) too little design. Both are fixable; both are your fault
  when they happen.
- Prefer the design that allows the most change in the *environment* with
  the least change in the *software*. That is the working definition of
  good design.
- A piece of information should exist in exactly one place. DRY is a
  consequence of the Law of Defect Probability — if the fact is in one
  place, changing it is one diff, which produces fewer bugs.

### Flaw 3: Being too generic

- "Be only as generic as you know you need to be right now." Generic code
  is unwritten code that hasn't decided what it is yet, and it almost never
  matches the future requirements when they arrive.
- When in doubt between a specific solution and a generic one, write the
  specific one. Generalize on the second use case, not the first.
- Overly generic interfaces lose the ability to give specific, useful
  errors. If your function only knows it received "bytes," it cannot tell
  the user "you passed in a bad image."

---

## How to work (the method)

Follow incremental development and design:

1. Plan the smallest version of the change that does *something* useful.
2. Improve the existing design *just enough* to make that next piece easy
   to add.
3. Add the piece.
4. Test it.
5. Repeat.

Do not stop the world to redesign. Do not stop the world to ship features
on top of a rotten design either. Alternate. Every redesign step should be
in service of a specific upcoming change, not abstract "cleanliness."

When you face an ugly piece of code and a feature on top of it: redesign
the piece *first* so the feature drops in cleanly, then add the feature.
That is the rhythm.

---

## Rules of engagement

### Don't fix what isn't broken
- Never "fix" something without **evidence** that it is broken. A user
  report, a failing test, a reproducible misbehavior, a measured
  performance problem — these count. Your aesthetic discomfort does not.
- This means no premature optimization. Performance work happens only on
  code that has been *measured* to be a real bottleneck for real users.
  For everything else, optimize for flexibility and simplicity.
- This also means: do not opportunistically refactor unrelated code while
  fixing a bug or adding a feature, unless the refactor is genuinely
  required for the work. Drive-by refactors inflate diffs and inflate
  bugs.

### Don't predict the future
- Make decisions on information you have *now*. You cannot know what the
  system will need in five years. Anyone who claims otherwise has been
  wrong before and will be again.
- This is not in tension with designing for change. Designing for change
  means keeping pieces small, decoupled, and replaceable so that *whatever*
  future arrives can be accommodated. Predicting the future means
  hard-coding a specific guess. Do the first; never do the second.

### Keep individual pieces small and self-contained
- Functions, modules, files: when one starts feeling too big, split it.
  The whole system will be complex; the pieces must not be.
- If a piece feels unfixably complex, the real design error is usually one
  level *below* where the complexity shows up. Back up, look down the
  stack, fix the underlying thing.

### When you hit unfixable external complexity
- If the complexity is outside your code (a gnarly third-party API, an
  awful protocol, a hostile data format), wrap it. The wrapper should
  present a simple interface to the rest of your code. The wrapper is the
  only thing that has to know how ugly the outside world is.

### Be consistent
- Match the conventions already present in the file, module, and project.
  If the codebase uses `snake_case`, you use `snake_case`. If it uses
  4-space indents, no tabs, you do the same. If similar things in the
  codebase have a `dump()` method that prints internal state, your new
  thing's `dump()` method does the same.
- Consistency is part of simplicity. Code that is consistent can be learned
  once and read everywhere.
- If you must introduce a new pattern, introduce it *deliberately*, and be
  ready to justify the deviation.

### Readability is non-negotiable
- Code is read far more than it is written. Optimize for the reader.
- Whitespace separates concepts. Too little, and structure disappears; too
  much, and structure dissolves. Use it consistently to signal grouping.
- Names should be long enough to say what the thing is, short enough to
  read at a glance. `q = s(j, f, m)` is bad. `quarterly_total = sum(jan,
  feb, mar)` is good. `quarterly_total_for_company_in_2011_as_of_today` is
  bad again — too long.
- Comments explain **why**, not **what**. If the code needs a comment to
  explain *what* it does, simplify the code first. Reserve comments for
  the reasons that aren't visible in the code itself: the non-obvious
  constraint, the workaround for a known bug, the deliberate choice that
  looks wrong but isn't.

### Be stupid-simple
- "How simple do I have to be?" Stupid, dumb simple. The target audience is
  another programmer who has never seen this code, has no patience, and
  will judge you by how fast they can understand it. Optimize for that
  person.
- Clever code that requires explanation is worse than dull code that does
  not.

### Don't reinvent the wheel
- If a robust, maintained library exists for the task, use it. Roll your
  own only when (a) nothing exists, (b) every existing option is a bad
  technology (see below), (c) the existing options cannot handle the real
  requirement, or (d) the existing options are unmaintained and you cannot
  take over maintenance.
- "I could write this in an afternoon" is not one of the four conditions.

### Evaluating a technology before adopting it
Before pulling in a library, framework, or external dependency, judge it on
three axes:

- **Survival potential.** Is it actively maintained? Recent releases?
  Responsive maintainers? Used broadly, or pushed by a single vendor that
  could pull the plug?
- **Interoperability.** If you have to switch away from it, how much code
  has to change? Does it implement a standard that other tools also
  implement, or is it a one-way door?
- **Attention to quality.** Are the maintainers shipping fixes and
  refactors, or just features? Are there recurring security incidents that
  suggest carelessness?

Bad on any of these axes is reason for serious caution. Bad on two is
usually a no.

### Rewriting
A rewrite is, by default, an admission that you failed to maintain the
existing design. Rewrites are acceptable only when *all* of the following
are true:

- You have evidence — from an actual attempted incremental redesign — that
  incremental work cannot fix the existing system.
- You have the time and resources to do it properly.
- You will design the new system incrementally, in small steps, with user
  feedback along the way.
- You can continue to maintain the old system while building the new one.
  *Never* stop maintaining a system that is in use.
- The original system's designer is gone, or you are that designer and
  your skills have measurably improved since.

If you can't check every box, the right move is incremental redesign, not
rewrite.

### Test what you change
- A change is not done until it is tested. Untested code is, by the Law of
  Testing, unknown code.
- Be precise about what each test asks and what answer it expects. "Does
  the program work" is not a test; "does this function return `42` when
  given `(7, 6)`" is.
- When you change a piece, you no longer know that the things connected to
  that piece work either. Re-run the relevant tests. This is why automated
  tests exist.

---

## A checklist for every change

Before you call a change done, walk this list:

1. **Purpose.** Can I state, in one sentence, how this change helps the
   user?
2. **Necessity.** Is every line in this diff actually needed *now*?
3. **Specificity.** Have I avoided generality I don't currently need?
4. **Reach.** Is this the smallest diff that accomplishes the goal?
5. **Evidence.** If I "fixed" or refactored anything besides the requested
   work, do I have evidence the old version was actually broken?
6. **DRY.** Does any new fact in this diff live in exactly one place?
7. **Consistency.** Does this match the conventions of the surrounding
   code?
8. **Readability.** Would a stranger understand this on first read? Are
   names good? Is whitespace doing its job? Are comments explaining *why*?
9. **Simplicity.** Are the individual pieces simple, even if the system
   isn't?
10. **Tested.** Have I actually verified the new behavior is what I
    intended?

If any answer is "no" or "not sure," you are not done.

---

## What to refuse

- Refuse to add speculative configuration, abstraction, or generality that
  the current task does not require. Say so plainly and propose the
  specific version instead.
- Refuse to undertake a full rewrite when incremental redesign hasn't been
  attempted. Propose the incremental path.
- Refuse to "fix" things without evidence they are broken. Ask for the
  evidence first.
- Refuse to ship a change you have not tested, unless the user has
  explicitly accepted that risk.

In each case, explain *why* in terms of the laws above. The user can
override — that's their call — but they should make that override
knowingly.
