package quotes

// NewStoics returns a Source of hand-curated lines from the Stoic
// canon — Marcus Aurelius (*Meditations*), Seneca (*Letters from a
// Stoic*), and Epictetus (*Discourses* / *Enchiridion*). All sources
// are public domain. Each entry carries its own trailing
// " — Author" attribution.
func NewStoics() *Source { return newSource("Stoics", stoics) }

var stoics = []string{
	// Marcus Aurelius — Meditations
	"Waste no more time arguing what a good man should be. Be one. — Marcus Aurelius",
	"You have power over your mind — not outside events. Realize this, and you will find strength. — Marcus Aurelius",
	"The happiness of your life depends upon the quality of your thoughts. — Marcus Aurelius",
	"If it is not right, do not do it; if it is not true, do not say it. — Marcus Aurelius",
	"When you arise in the morning, think of what a precious privilege it is to be alive — to breathe, to think, to enjoy, to love. — Marcus Aurelius",
	"The best revenge is to be unlike him who performed the injury. — Marcus Aurelius",
	"Everything we hear is an opinion, not a fact. Everything we see is a perspective, not the truth. — Marcus Aurelius",
	"Confine yourself to the present. — Marcus Aurelius",
	"The impediment to action advances action. What stands in the way becomes the way. — Marcus Aurelius",
	"How much trouble he avoids who does not look to see what his neighbor says or does. — Marcus Aurelius",
	"Accept the things to which fate binds you, and love the people with whom fate brings you together, and do so with all your heart. — Marcus Aurelius",
	"Loss is nothing else but change, and change is Nature's delight. — Marcus Aurelius",
	"It is not death that a man should fear, but he should fear never beginning to live. — Marcus Aurelius",
	"Look back over the past, with its changing empires that rose and fell, and you can foresee the future too. — Marcus Aurelius",
	"Dwell on the beauty of life. Watch the stars, and see yourself running with them. — Marcus Aurelius",
	"Very little is needed to make a happy life; it is all within yourself, in your way of thinking. — Marcus Aurelius",

	// Seneca — Letters from a Stoic, On the Shortness of Life
	"We suffer more often in imagination than in reality. — Seneca",
	"It is not that we have a short time to live, but that we waste a lot of it. — Seneca",
	"As is a tale, so is life: not how long it is, but how good it is, is what matters. — Seneca",
	"Luck is what happens when preparation meets opportunity. — Seneca",
	"Difficulties strengthen the mind, as labor does the body. — Seneca",
	"He who is brave is free. — Seneca",
	"Begin at once to live, and count each separate day as a separate life. — Seneca",
	"If a man knows not to which port he sails, no wind is favorable. — Seneca",
	"Sometimes even to live is an act of courage. — Seneca",
	"Wherever there is a human being, there is an opportunity for a kindness. — Seneca",
	"Until we have begun to go without them, we fail to realize how unnecessary many things are. — Seneca",
	"No man was ever wise by chance. — Seneca",
	"There is no genius without a touch of madness. — Seneca",
	"It does not matter how many books you have, but how good are the books which you have. — Seneca",
	"He suffers more than necessary, who suffers before it is necessary. — Seneca",

	// Epictetus — Enchiridion, Discourses
	"It's not what happens to you, but how you react to it that matters. — Epictetus",
	"Wealth consists not in having great possessions, but in having few wants. — Epictetus",
	"First say to yourself what you would be; and then do what you have to do. — Epictetus",
	"No man is free who is not master of himself. — Epictetus",
	"We have two ears and one mouth so that we can listen twice as much as we speak. — Epictetus",
	"Don't explain your philosophy. Embody it. — Epictetus",
	"He is a wise man who does not grieve for the things which he has not, but rejoices for those which he has. — Epictetus",
	"Only the educated are free. — Epictetus",
	"If you want to improve, be content to be thought foolish and stupid. — Epictetus",
	"Make the best use of what is in your power, and take the rest as it happens. — Epictetus",
	"Circumstances don't make the man, they only reveal him to himself. — Epictetus",
	"Any person capable of angering you becomes your master. — Epictetus",
	"It is impossible for a man to learn what he thinks he already knows. — Epictetus",
	"Freedom is the only worthy goal in life. It is won by disregarding things that lie beyond our control. — Epictetus",
	"First learn the meaning of what you say, and then speak. — Epictetus",
}
