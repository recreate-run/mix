import { cn } from '@/lib/utils';

function IntroHeader({
  className,
  children,
  ...props
}: React.ComponentProps<'section'>) {
  return (
    <section className={cn('border-grid', className)} {...props}>
      <div className="container-wrapper">
        <div className="container flex flex-col items-center gap-2 text-center xl:gap-4">
          {children}
        </div>
      </div>
    </section>
  );
}

function IntroHeaderHeading({
  className,
  ...props
}: React.ComponentProps<'h1'>) {
  return (
    <h1
      className={cn(
        'max-w-2xl text-balance font-semibold text-8xl text-primary leading-tighter tracking-tight lg:font-semibold lg:leading-[1.1] xl:tracking-tighter',
        className
      )}
      {...props}
    />
  );
}

function IntroHeaderDescription({
  className,
  ...props
}: React.ComponentProps<'p'>) {
  return (
    <p
      className={cn(
        'max-w-3xl text-balance text-base text-muted-foreground sm:text-xl',
        className
      )}
      {...props}
    />
  );
}

function IntroActions({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      className={cn(
        'flex w-full items-center justify-center gap-2 pt-2 **:data-[slot=button]:shadow-none',
        className
      )}
      {...props}
    />
  );
}

export {
  IntroActions,
  IntroHeader,
  IntroHeaderDescription,
  IntroHeaderHeading,
};
