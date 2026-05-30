'use client';

import { useEffect, useId, useMemo, useRef, useState } from 'react';
import { ImagePlus, X } from 'lucide-react';
import { cn } from '@/shared/lib/utils';

interface PhotoUploadProps {
  label: string;
  value: File | null;
  onChange: (file: File | null) => void;
  required?: boolean;
  /** Accepted MIME types (default images only). */
  accept?: string;
  /** Max size in MB (default 5). */
  maxSizeMB?: number;
  error?: string;
}

// PhotoUpload is an accessible single-image dropzone with live preview.
export function PhotoUpload({
  label,
  value,
  onChange,
  required,
  accept = 'image/png,image/jpeg,image/webp',
  maxSizeMB = 5,
  error,
}: PhotoUploadProps) {
  const inputId = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);

  // Derive the preview URL from the file; revoke the previous URL on change/unmount.
  const preview = useMemo(() => (value ? URL.createObjectURL(value) : null), [value]);
  useEffect(() => {
    if (!preview) return;
    return () => URL.revokeObjectURL(preview);
  }, [preview]);

  const accept_ = accept.split(',');

  const pick = (file: File | null) => {
    setLocalError(null);
    if (!file) return onChange(null);
    if (!accept_.includes(file.type)) {
      setLocalError('Chỉ chấp nhận ảnh PNG, JPG hoặc WEBP.');
      return;
    }
    if (file.size > maxSizeMB * 1024 * 1024) {
      setLocalError(`Ảnh vượt quá ${maxSizeMB}MB.`);
      return;
    }
    onChange(file);
  };

  const shownError = error ?? localError;

  return (
    <div className="space-y-1.5">
      <label htmlFor={inputId} className="text-foreground text-sm font-medium">
        {label}
        {required && <span className="text-destructive ml-0.5">*</span>}
      </label>

      <div
        role="button"
        tabIndex={0}
        aria-describedby={shownError ? `${inputId}-error` : undefined}
        onClick={() => inputRef.current?.click()}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            inputRef.current?.click();
          }
        }}
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => {
          e.preventDefault();
          setDragging(false);
          pick(e.dataTransfer.files?.[0] ?? null);
        }}
        className={cn(
          'focus-visible:ring-ring relative flex min-h-40 cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed p-4 text-center transition-colors duration-200 focus-visible:ring-2 focus-visible:outline-none',
          dragging ? 'border-primary bg-primary/5' : 'border-input hover:border-primary/60 hover:bg-accent/40',
          shownError && 'border-destructive',
        )}
      >
        {preview ? (
          <>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={preview} alt={`Xem trước ${label}`} className="max-h-36 rounded-md object-contain" />
            <button
              type="button"
              aria-label={`Xoá ${label}`}
              onClick={(e) => {
                e.stopPropagation();
                pick(null);
                if (inputRef.current) inputRef.current.value = '';
              }}
              className="bg-background/90 text-foreground hover:bg-background absolute top-2 right-2 inline-flex h-8 w-8 items-center justify-center rounded-full border shadow-sm transition-colors"
            >
              <X className="h-4 w-4" />
            </button>
          </>
        ) : (
          <>
            <ImagePlus className="text-muted-foreground h-8 w-8" aria-hidden="true" />
            <p className="text-muted-foreground text-sm">
              Kéo thả ảnh vào đây, hoặc <span className="text-primary font-medium">chọn tệp</span>
            </p>
            <p className="text-muted-foreground/70 text-xs">PNG, JPG, WEBP · tối đa {maxSizeMB}MB</p>
          </>
        )}

        <input
          ref={inputRef}
          id={inputId}
          type="file"
          accept={accept}
          className="sr-only"
          onChange={(e) => pick(e.target.files?.[0] ?? null)}
        />
      </div>

      {shownError && (
        <p id={`${inputId}-error`} role="alert" className="text-destructive text-sm">
          {shownError}
        </p>
      )}
    </div>
  );
}
