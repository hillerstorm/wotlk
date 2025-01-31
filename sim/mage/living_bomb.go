package mage

import (
	"strconv"
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (mage *Mage) registerLivingBombSpell() {
	baseCost := .22 * mage.BaseMana
	bonusCrit := float64(mage.Talents.WorldInFlames+mage.Talents.CriticalMass) * 2 * core.CritRatingPerCritChance

	livingBombExplosionSpell := mage.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 55362},
		SpellSchool: core.SpellSchoolFire,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       SpellFlagMage | HotStreakSpells,

		BonusCritRating:  bonusCrit,
		DamageMultiplier: mage.spellDamageMultiplier,
		CritMultiplier:   mage.SpellCritMultiplier(1, mage.bonusCritDamage),
		ThreatMultiplier: 1 - 0.1*float64(mage.Talents.BurningSoul),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := 690 + 0.4*spell.SpellPower()
			baseDamage *= sim.Encounter.AOECapMultiplier()
			for _, aoeTarget := range sim.Encounter.Targets {
				spell.CalcAndDealDamage(sim, &aoeTarget.Unit, baseDamage, spell.OutcomeMagicHitAndCrit)
			}
		},
	})

	mage.LivingBomb = mage.RegisterSpell(core.SpellConfig{
		ActionID:     core.ActionID{SpellID: 55360},
		SpellSchool:  core.SpellSchoolFire,
		ProcMask:     core.ProcMaskSpellDamage,
		Flags:        SpellFlagMage,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost,
				GCD:  core.GCDDefault,
			},
		},

		BonusCritRating:  bonusCrit,
		DamageMultiplier: mage.spellDamageMultiplier,
		CritMultiplier:   mage.SpellCritMultiplier(1, mage.bonusCritDamage),
		ThreatMultiplier: 1 - 0.1*float64(mage.Talents.BurningSoul),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				mage.LivingBombDot.Apply(sim)
			}
			spell.DealOutcome(sim, result)
		},
	})

	target := mage.CurrentTarget

	onTick := func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
		dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
	}
	if mage.HasMajorGlyph(proto.MageMajorGlyph_GlyphOfLivingBomb) {
		onTick = func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeSnapshotCrit)
		}
	}

	mage.LivingBombDot = core.NewDot(core.Dot{
		Spell: mage.LivingBomb,
		Aura: target.RegisterAura(core.Aura{
			Label:    "LivingBomb-" + strconv.Itoa(int(mage.Index)),
			ActionID: mage.LivingBomb.ActionID,
			Tag:      "LivingBomb",
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				livingBombExplosionSpell.Cast(sim, target)
			},
		}),

		NumberOfTicks: 4,
		TickLength:    time.Second * 3,

		OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, _ bool) {
			dot.SnapshotBaseDamage = 345 + 0.2*dot.Spell.SpellPower()
			dot.SnapshotCritChance = dot.Spell.SpellCritChance(target)
			dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(dot.Spell.Unit.AttackTables[target.UnitIndex])
		},
		OnTick: onTick,
	})
}
