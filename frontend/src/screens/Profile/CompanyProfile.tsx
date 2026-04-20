import { useState, useRef } from "react";
import { useI18n } from "../../providers/useI18n";
import { Field } from "../../ui/Field";
import { Button } from "../../ui/Button";
import { CheckIcon, XIcon } from "../../icons/Icons";
import { setItem } from "../../lib/storage";
import { loadProfile, PROFILE_STORAGE_KEY } from "./profileData";
import type { CompanyProfileData } from "./profileData";
import styles from "./CompanyProfile.module.css";

const CERT_KEYS = ["cert_iso27001", "cert_soc2", "cert_152fz", "cert_pci", "cert_gdpr"] as const;
const SPEC_KEYS = ["spec_web", "spec_mobile", "spec_integrations", "spec_ml", "spec_devops", "spec_security", "spec_enterprise", "spec_ecommerce"] as const;

export function CompanyProfile() {
  const { t } = useI18n();
  const saved = loadProfile();

  const [name, setName] = useState(saved.name);
  const [teamSize, setTeamSize] = useState(saved.teamSize);
  const [experience, setExperience] = useState(saved.experience);
  const [stack, setStack] = useState<string[]>(saved.stack);
  const [certs, setCerts] = useState<string[]>(saved.certs);
  const [specializations, setSpecializations] = useState<string[]>(saved.specializations);
  const [clients, setClients] = useState(saved.clients);
  const [extra, setExtra] = useState(saved.extra);
  const [showSaved, setShowSaved] = useState(false);
  const [tagInput, setTagInput] = useState("");
  const tagRef = useRef<HTMLInputElement>(null);

  function toggleItem(list: string[], item: string): string[] {
    return list.includes(item) ? list.filter((i) => i !== item) : [...list, item];
  }

  function addTag() {
    const val = tagInput.trim();
    if (val && !stack.includes(val)) {
      setStack([...stack, val]);
    }
    setTagInput("");
  }

  function handleTagKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      addTag();
    }
    if (e.key === "Backspace" && tagInput === "" && stack.length > 0) {
      setStack(stack.slice(0, -1));
    }
  }

  function handleSave() {
    const data: CompanyProfileData = {
      name, teamSize, experience, stack, certs, specializations, clients, extra,
    };
    setItem(PROFILE_STORAGE_KEY, data);
    setShowSaved(true);
    setTimeout(() => setShowSaved(false), 2000);
  }

  return (
    <div className={`screen-enter ${styles.screen}`}>
      {/* Header */}
      <div className={styles.header}>
        <h2 className={`t-h2 font-serif ${styles.title}`}>{t.profile.title}</h2>
        <p className={`t-body ${styles.subtitle}`}>{t.profile.subtitle}</p>
      </div>

      {/* Basic info */}
      <div className={styles.section}>
        <div className={styles.row}>
          <Field label={t.profile.name} tooltip={t.profile.name_tip}>
            <input
              className={styles.input}
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </Field>
          <div className={styles.row}>
            <Field label={t.profile.team_size} tooltip={t.profile.team_size_tip}>
              <input
                className={styles.inputSmall}
                type="number"
                min="1"
                value={teamSize}
                onChange={(e) => setTeamSize(e.target.value)}
              />
            </Field>
            <Field label={t.profile.experience} tooltip={t.profile.experience_tip}>
              <div className={styles.inputSuffix}>
                <input
                  className={styles.inputSmall}
                  type="number"
                  min="0"
                  value={experience}
                  onChange={(e) => setExperience(e.target.value)}
                />
                <span className={`t-small ${styles.suffixLabel}`}>{t.profile.experience_suffix}</span>
              </div>
            </Field>
          </div>
        </div>
      </div>

      {/* Tech stack tags */}
      <div className={styles.section}>
        <Field label={t.profile.stack} tooltip={t.profile.stack_tip}>
          <div className={styles.tagsWrap} onClick={() => tagRef.current?.focus()}>
            {stack.map((tag) => (
              <span key={tag} className={styles.tag}>
                {tag}
                <button
                  type="button"
                  className={styles.tagRemove}
                  onClick={(e) => { e.stopPropagation(); setStack(stack.filter((s) => s !== tag)); }}
                  aria-label={`Remove ${tag}`}
                >
                  <XIcon />
                </button>
              </span>
            ))}
            <input
              ref={tagRef}
              className={styles.tagInput}
              type="text"
              value={tagInput}
              onChange={(e) => setTagInput(e.target.value)}
              onKeyDown={handleTagKeyDown}
              onBlur={addTag}
              placeholder={stack.length === 0 ? t.profile.stack_placeholder : ""}
            />
          </div>
        </Field>
      </div>

      {/* Certifications */}
      <div className={styles.section}>
        <Field label={t.profile.certs} tooltip={t.profile.certs_tip}>
          <div className={styles.checkGrid}>
            {CERT_KEYS.map((key) => {
              const checked = certs.includes(key);
              return (
                <label key={key} className={styles.checkItem}>
                  <span
                    className={checked ? styles.checkboxChecked : styles.checkbox}
                    onClick={() => setCerts(toggleItem(certs, key))}
                    role="checkbox"
                    aria-checked={checked}
                  >
                    {checked && <CheckIcon />}
                  </span>
                  <span className={styles.checkLabel}>{t.profile[key]}</span>
                </label>
              );
            })}
          </div>
        </Field>
      </div>

      {/* Specializations */}
      <div className={styles.section}>
        <Field label={t.profile.specializations} tooltip={t.profile.specializations_tip}>
          <div className={styles.checkGrid}>
            {SPEC_KEYS.map((key) => {
              const checked = specializations.includes(key);
              return (
                <label key={key} className={styles.checkItem}>
                  <span
                    className={checked ? styles.checkboxChecked : styles.checkbox}
                    onClick={() => setSpecializations(toggleItem(specializations, key))}
                    role="checkbox"
                    aria-checked={checked}
                  >
                    {checked && <CheckIcon />}
                  </span>
                  <span className={styles.checkLabel}>{t.profile[key]}</span>
                </label>
              );
            })}
          </div>
        </Field>
      </div>

      {/* Clients */}
      <div className={styles.section}>
        <Field label={t.profile.clients} tooltip={t.profile.clients_tip}>
          <textarea
            className={styles.textarea}
            value={clients}
            onChange={(e) => setClients(e.target.value)}
            placeholder={t.profile.clients_placeholder}
            rows={3}
          />
        </Field>
      </div>

      {/* Extra */}
      <div className={styles.section}>
        <Field label={t.profile.extra} tooltip={t.profile.extra_tip}>
          <textarea
            className={styles.textarea}
            value={extra}
            onChange={(e) => setExtra(e.target.value)}
            placeholder={t.profile.extra_placeholder}
            rows={3}
          />
        </Field>
      </div>

      {/* Save */}
      <div className={styles.footer}>
        <Button variant="brand" size="lg" onClick={handleSave}>
          {t.profile.save}
        </Button>
        {showSaved && (
          <span className={`t-small ${styles.savedMsg}`}>
            <CheckIcon /> {t.profile.saved}
          </span>
        )}
      </div>
    </div>
  );
}
