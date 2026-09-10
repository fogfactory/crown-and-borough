# Persona catalogue

Each bot chooses one base persona and may override `play_style`, `trust`, and
`tone` before its first order. The YAML files are human-readable configuration;
`persona-init.sh` persists the resolved JSON used by the runtime.

## Traits

- `play_style`: `aggressive`, `defensive`, `opportunistic`, `mercantile`,
  `honest`, `treacherous`, or `random`;
- `trust`: `high`, `medium`, `low`, or `none`;
- `tone`: `curt`, `formal`, `friendly`, `theatrical`, or `threatening`.

The dimensions are independent. A defensive player can be treacherous. A
mercantile player can have no trust. The runtime prompt must preserve those
combinations instead of collapsing them into a generic diplomat.

## Catalogue

| Id | Core behavior | Default trust | Useful test |
|---|---|---:|---|
| `conqueror` | Expand early, pressure the weakest border, accept only profitable deals. | none | Aggression, combat, retaliation |
| `diplomat` | Build temporary coalitions and trade commitments for time. | medium | Negotiation and coalition changes |
| `turtle` | Protect capital, improve supply, attack only with a clear advantage. | high | Long games and economic pressure |
| `brigand` | Pillage exposed infrastructure and attack isolated armies. | none | Raids and supply disruption |
| `merchant` | Accumulate resources, fund targeted gifts, and buy non-aggression. | medium | Winter economy and transfers |
| `honorable` | Keep explicit pacts unless the other side breaks them first. | high | Trust baseline |
| `intriguer` | Make asymmetric promises, conceal the main target, and betray on advantage. | low | Deception and information asymmetry |
| `chaotic` | Vary openings and targets while preserving legality. | low | Robustness against unpredictable orders |

Never choose a random persona after negotiation starts. Reproducible persona
assignment makes parallel game failures diagnosable.
