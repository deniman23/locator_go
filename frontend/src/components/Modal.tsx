import React, { useEffect, useId, useRef } from 'react';

type ModalProps = {
    title: string;
    onClose: () => void;
    children: React.ReactNode;
    footer?: React.ReactNode;
    /** When true, backdrop click does not close (e.g. one-time API key). */
    disableBackdropClose?: boolean;
    /** When true, Escape does not call onClose. */
    disableEscapeClose?: boolean;
    /** Hide header × (force close via footer only). */
    hideCloseButton?: boolean;
    className?: string;
    contentClassName?: string;
};

const FOCUSABLE =
    'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

const Modal: React.FC<ModalProps> = ({
    title,
    onClose,
    children,
    footer,
    disableBackdropClose = false,
    disableEscapeClose = false,
    hideCloseButton = false,
    className = '',
    contentClassName = '',
}) => {
    const titleId = useId();
    const panelRef = useRef<HTMLDivElement>(null);
    const previousFocus = useRef<HTMLElement | null>(null);
    const onCloseRef = useRef(onClose);
    onCloseRef.current = onClose;

    useEffect(() => {
        previousFocus.current = document.activeElement as HTMLElement | null;
        const panel = panelRef.current;
        if (panel) {
            const focusable = panel.querySelectorAll<HTMLElement>(FOCUSABLE);
            (focusable[0] ?? panel).focus();
        }

        const onKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                e.preventDefault();
                if (!disableEscapeClose) onCloseRef.current();
                return;
            }
            if (e.key !== 'Tab' || !panelRef.current) return;
            const focusable = Array.from(panelRef.current.querySelectorAll<HTMLElement>(FOCUSABLE));
            if (focusable.length === 0) {
                e.preventDefault();
                return;
            }
            const first = focusable[0];
            const last = focusable[focusable.length - 1];
            if (e.shiftKey && document.activeElement === first) {
                e.preventDefault();
                last.focus();
            } else if (!e.shiftKey && document.activeElement === last) {
                e.preventDefault();
                first.focus();
            }
        };

        document.addEventListener('keydown', onKeyDown);
        return () => {
            document.removeEventListener('keydown', onKeyDown);
            previousFocus.current?.focus?.();
        };
    }, [disableEscapeClose]);

    return (
        <div
            className={`modal-backdrop ${className}`.trim()}
            role="presentation"
            onMouseDown={(e) => {
                if (disableBackdropClose) return;
                if (e.target === e.currentTarget) onClose();
            }}
        >
            <div
                ref={panelRef}
                className={`modal-panel ${contentClassName}`.trim()}
                role="dialog"
                aria-modal="true"
                aria-labelledby={titleId}
                tabIndex={-1}
            >
                <div className="modal-header">
                    <h3 id={titleId}>{title}</h3>
                    {!hideCloseButton && (
                        <button type="button" className="close-button" onClick={onClose} aria-label="Закрыть">
                            ×
                        </button>
                    )}
                </div>
                <div className="modal-body">{children}</div>
                {footer && <div className="modal-footer">{footer}</div>}
            </div>
        </div>
    );
};

export default Modal;
