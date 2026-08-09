import type { Season } from '@/types'

export interface OrderReferenceExample {
  syntax: string
  description: string
}

export interface OrderReferenceEntry {
  symbol: string
  name: string
  syntax: string
  condition: string
  scope: string
  cost: number | null
  result: string
  examples: readonly OrderReferenceExample[]
}

export interface OrderReferenceNote {
  title: string
  description: string
  examples?: readonly OrderReferenceExample[]
}

export const ACTION_ORDER_REFERENCES: readonly OrderReferenceEntry[] = [
  {
    symbol: 'A',
    name: 'Attaque / déplacement',
    syntax: 'XXX A YYY',
    condition: 'YYY doit être adjacent à XXX par une frontière franchissable.',
    scope: "L'armée entière se déplace vers YYY et peut y combattre une armée ennemie.",
    cost: null,
    result:
      "L'armée avance si son attaque est la plus haute force strictement unique ; sinon elle reste sur place.",
    examples: [
      {
        syntax: 'BRI A ATL',
        description: 'Attaquer ou occuper ATL depuis BRI.',
      },
    ],
  },
  {
    symbol: 'S',
    name: 'Soutien',
    syntax: 'XXX S YYY [- ZZZ]',
    condition:
      'YYY est la case soutenue ; en soutien offensif, ZZZ est la destination de son attaque.',
    scope:
      "Renforce une armée de n'importe quelle nationalité, en défense ou pour une attaque précise.",
    cost: null,
    result:
      "Le soutien ne compte que si l'armée soutenue accomplit l'action annoncée ; une attaque peut le couper.",
    examples: [
      {
        syntax: 'BRI S ATL',
        description: "Soutien défensif de l'armée qui tient ATL.",
      },
      {
        syntax: 'BRI S ATL - NOR',
        description: "Soutien offensif de l'attaque d'ATL vers NOR.",
      },
    ],
  },
  {
    symbol: 'H',
    name: 'Maintien',
    syntax: 'H XXX',
    condition: 'Aucune cible supplémentaire.',
    scope: "L'armée reste sur XXX et conserve sa position.",
    cost: null,
    result: "L'armée ne se déplace pas et peut recevoir un soutien défensif.",
    examples: [
      {
        syntax: 'H BRI',
        description: 'Maintenir la position sur BRI.',
      },
    ],
  },
  {
    symbol: 'J',
    name: 'Jonction',
    syntax: 'XXX J YYY',
    condition:
      'YYY doit être adjacent à XXX ; J doit être le dernier ordre de la chaîne.',
    scope: "Déplacement pacifique de l'armée entière vers YYY.",
    cost: null,
    result: 'La jonction ne combat pas et est repoussée si la destination est contestée.',
    examples: [
      {
        syntax: 'BRI J ROS',
        description: 'Rejoindre ROS sans lancer de combat.',
      },
    ],
  },
  {
    symbol: 'P',
    name: 'Pillage',
    syntax: 'P XXX',
    condition: 'Une infrastructure doit être présente sur XXX.',
    scope: "Détruit l'infrastructure de la case occupée.",
    cost: null,
    result:
      'Le bonus de pillage est crédité à la source alliée la plus proche et peut réduire une famine.',
    examples: [
      {
        syntax: 'P BRI',
        description: "Piller l'infrastructure de BRI.",
      },
    ],
  },
  {
    symbol: 'O',
    name: 'Prise en otage',
    syntax: 'XXX O NNN',
    condition: "NNN doit être un noble prisonnier détenu sur la case de l'armée.",
    scope: 'Le statut du noble ciblé devient hostage (otage).',
    cost: null,
    result:
      "L'otage reste prisonnier ; il ne peut pas émettre de nouvelle chaîne tant qu'il est détenu.",
    examples: [
      {
        syntax: 'BRI O HUG',
        description: 'Détenir HUG comme otage sur BRI.',
      },
    ],
  },
  {
    symbol: 'K',
    name: 'Mise au donjon',
    syntax: 'XXX K NNN',
    condition: "NNN doit être un noble prisonnier détenu sur la case de l'armée.",
    scope: 'Le statut du noble ciblé devient dungeon (donjon).',
    cost: null,
    result:
      "Le noble est retiré de toute émission de chaîne jusqu'à sa libération en hiver.",
    examples: [
      {
        syntax: 'BRI K HUG',
        description: 'Placer HUG au donjon sur BRI.',
      },
    ],
  },
  {
    symbol: 'D',
    name: 'Dispersion',
    syntax: 'XXX D YYY ZZZ ...',
    condition:
      'Il faut une destination par troupe, adjacente à XXX ou égale à XXX, sans doublon, et affecter explicitement les nobles présents.',
    scope:
      "Sépare l'armée en une unité par destination ; * affecte tous les nobles non affectés, *NNN un noble précis.",
    cost: null,
    result:
      'Chaque groupe se déplace pacifiquement ; une destination contestée repousse le groupe concerné.',
    examples: [
      {
        syntax: 'BRI D ATL NOR',
        description: 'Deux destinations pour une armée de deux troupes.',
      },
      {
        syntax: 'BRI D ATL NOR BRU',
        description: 'Trois destinations distinctes pour trois troupes.',
      },
      {
        syntax: 'BRI D ATL*HUG NOR',
        description: 'Affecter HUG à ATL ; envoyer l’autre unité vers NOR.',
      },
      {
        syntax: 'BRI D ATL*HUG*JEA NOR*ROS',
        description: 'Affecter HUG et JEA à ATL, et ROS à NOR.',
      },
      {
        syntax: '(BRI D ATL NOR)',
        description: "Version loop : retenter jusqu'à la résolution intégrale.",
      },
    ],
  },
]

export const LIAISON_MODE_REFERENCES: readonly OrderReferenceNote[] = [
  {
    title: 'single',
    description:
      "Une ligne sans parenthèses : la chaîne s'arrête au premier échec, et son suffixe est abandonné.",
    examples: [
      {
        syntax: 'H BRI',
        description: 'Ordre exécuté une fois.',
      },
    ],
  },
  {
    title: 'loop',
    description:
      "Une ligne entre parenthèses : l'ordre est retenté jusqu'à sa réussite. Un maintien en loop met l'armée en veille ; une erreur mécaniquement impossible casse toujours la chaîne.",
    examples: [
      {
        syntax: '(H BRI)',
        description: 'Maintien répété jusqu’à réception d’une nouvelle chaîne.',
      },
      {
        syntax: '(BRI A NOR)',
        description: "Retenter l'attaque à chaque résolution.",
      },
    ],
  },
]

export const SPECIAL_ORDER_NOTES: readonly OrderReferenceNote[] = [
  {
    title: 'Dispersion',
    description:
      "En single, une dispersion peut progresser partiellement. En loop, elle est retentée jusqu'à ce que toutes les unités soient réparties.",
  },
  {
    title: 'Prisonniers',
    description:
      "O et K ciblent uniquement un noble prisonnier présent sur la case de l'armée. L NNN le libère en hiver ; il réapparaît libre dans sa capitale.",
    examples: [
      {
        syntax: 'L N HUG',
        description: 'Libérer le noble HUG depuis la phase d’hiver.',
      },
    ],
  },
]

export interface WinterInvestmentReference extends OrderReferenceEntry {
  cost: number
}

export const WINTER_INVESTMENT_REFERENCES: readonly WinterInvestmentReference[] = [
  {
    symbol: 'R N',
    name: 'Recruter un noble',
    syntax: 'R N XXX',
    condition:
      'XXX doit être contrôlé, porter une armée du joueur et un château ou un village.',
    scope: 'Ajoute un noble au joueur sur XXX.',
    cost: 2,
    result: 'Le noble recruté rejoint la réserve libre du joueur.',
    examples: [
      {
        syntax: 'R N ATL',
        description: 'Recruter un noble sur ATL.',
      },
    ],
  },
  {
    symbol: 'R T',
    name: 'Recruter une troupe',
    syntax: 'R T XXX',
    condition:
      'XXX doit être contrôlé et un noble libre du joueur doit être sur XXX ou adjacent par une frontière franchissable.',
    scope:
      "Ajoute une troupe à l'armée du joueur sur XXX, ou crée une armée si XXX est vide.",
    cost: 1,
    result: 'Une troupe est ajoutée si le paiement est possible.',
    examples: [
      {
        syntax: 'R T ATL',
        description: 'Recruter une troupe sur ATL.',
      },
    ],
  },
  {
    symbol: 'C M',
    name: 'Construire ou améliorer un moulin',
    syntax: 'C M XXX',
    condition:
      'XXX doit être contrôlé ; un moulin doit respecter les conditions de structure.',
    scope: 'Construit ou améliore le moulin de XXX.',
    cost: 3,
    result: 'Le niveau du moulin augmente sa production stockable associée.',
    examples: [
      {
        syntax: 'C M ATL',
        description: 'Construire ou améliorer le moulin sur ATL.',
      },
    ],
  },
  {
    symbol: 'C C',
    name: 'Construire un château',
    syntax: 'C C XXX',
    condition: 'XXX doit être contrôlé.',
    scope: 'Construit un château sur XXX.',
    cost: 10,
    result:
      'Un village peut être remplacé par le château ; le stock de la case est conservé.',
    examples: [
      {
        syntax: 'C C ATL',
        description: 'Construire un château sur ATL.',
      },
    ],
  },
  {
    symbol: 'C D',
    name: 'Construire un dépôt de vivres',
    syntax: 'C D XXX',
    condition: 'XXX doit être contrôlé.',
    scope: 'Construit un dépôt de vivres sur XXX.',
    cost: 3,
    result: 'Le dépôt contrôlé augmente de deux cases la portée du ravitaillement.',
    examples: [
      {
        syntax: 'C D ATL',
        description: 'Construire un dépôt de vivres sur ATL.',
      },
    ],
  },
  {
    symbol: 'E C',
    name: 'Désigner une capitale',
    syntax: 'E C XXX',
    condition: 'XXX doit être un château contrôlé par le joueur.',
    scope: 'Change la capitale du joueur pour le château de XXX.',
    cost: 0,
    result:
      'Les futurs rapatriements de stock et les libérations utilisent cette capitale.',
    examples: [
      {
        syntax: 'E C ATL',
        description: 'Désigner le château d’ATL comme capitale.',
      },
    ],
  },
  {
    symbol: 'L N',
    name: 'Libérer un noble',
    syntax: 'L N XXX',
    condition:
      'XXX doit être le code du noble détenu par le joueur, et une capitale doit exister.',
    scope: 'Retire le statut hostage ou dungeon du noble ciblé.',
    cost: 0,
    result: 'Le noble réapparaît libre dans la capitale de son propriétaire.',
    examples: [
      {
        syntax: 'L N HUG',
        description: 'Libérer le noble HUG.',
      },
    ],
  },
]

export const WINTER_REFERENCE_NOTES: readonly OrderReferenceNote[] = [
  {
    title: 'Règle commune',
    description:
      "Tous les investissements exigent le contrôle du territoire ciblé. Le coût est prélevé d'abord sur son stock, puis sur la source contrôlée la plus proche ; aucun paiement partiel n'est effectué.",
  },
  {
    title: 'Fin de l’hiver',
    description:
      'Les stocks restants sont conservés à hauteur de ceil(stock / 2), puis rapatriés vers la capitale en laissant au maximum 1 R par village et 2 R par château hors capitale.',
  },
]

export function isActionSeason(season: Season): boolean {
  return season !== 'winter'
}
